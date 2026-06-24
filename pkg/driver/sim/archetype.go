package sim

import (
	"fmt"
	"math"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/sim/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/sim/scale"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/fanspeedpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/mockpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/motionsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

// Supported archetype types.
const (
	ArchetypeLighting   = "lighting"
	ArchetypeFCU        = "fcu"
	ArchetypePIR        = "pir"
	ArchetypeOccupancy  = "occupancy"
	ArchetypeMotion     = "motion"
	ArchetypeBrightness = "brightness"
	ArchetypeAirQuality = "airquality"
	ArchetypeMeter      = "meter"
	ArchetypeElectric   = "electric"
	ArchetypeEnterLeave = "enterleave"
)

// Per-device defaults at full load, in watts.
const (
	defaultLightingW = 60  // a ceiling LED panel
	defaultFcuW      = 400 // a fan coil unit at full fan
	defaultSetPointC = 21
)

// Device is an announce-ready expanded device.
type Device struct {
	Name     string
	Metadata *metadatapb.Metadata
	Features []node.Feature
}

// Expand turns the building config into a coupled simulation engine plus the list
// of devices to announce. Device updaters are registered on the returned Building.
// start is the simulated time at which the building begins.
func Expand(cfg config.Root, scaler scale.Time, start time.Time) (*Building, []Device, error) {
	// Build the floor/room state first; both the engine and the per-device updaters
	// need the room pointers. Each room is kept alongside its config so the
	// expansion pass below can use the pointer directly instead of re-indexing the
	// config positionally (which would silently mismatch if rooms were reordered).
	type roomBuild struct {
		fc   config.Floor
		rc   config.Room
		room *Room
	}
	floors := make([]*Floor, 0, len(cfg.Floors))
	var built []roomBuild
	for _, fc := range cfg.Floors {
		floor := &Floor{Name: fc.Name}
		roomMax := sharedOccupancy(fc)
		for _, rc := range fc.Rooms {
			room := &Room{
				Name:         rc.Name,
				MaxOccupancy: roomMax(rc),
				SetPointC:    defaultSetPointC,
			}
			for _, a := range rc.Archetypes {
				switch a.Type {
				case ArchetypeLighting:
					room.ratedLightingW += float64(count(a)) * ratedPower(a, defaultLightingW)
				case ArchetypeFCU:
					room.ratedFcuW += float64(count(a)) * ratedPower(a, defaultFcuW)
					if a.SetPointC > 0 {
						room.SetPointC = a.SetPointC
					}
				}
			}
			floor.Rooms = append(floor.Rooms, room)
			built = append(built, roomBuild{fc: fc, rc: rc, room: room})
		}
		floors = append(floors, floor)
	}

	b := NewBuilding(start, scaler, cfg.BaseLoadW, cfg.Seed, floors)
	ctrl := newController(b)

	// Device numbering is keyed by location+type rather than per archetype entry,
	// so a room listing the same type twice (e.g. to mix titles or rated powers)
	// continues lighting-01, -02, -03 instead of generating colliding names.
	counters := make(map[string]int)
	var devices []Device
	for _, rb := range built {
		for _, a := range rb.rc.Archetypes {
			ds, err := expandArchetype(cfg.NamePrefix, b, ctrl, rb.fc, rb.rc, rb.room, a, counters)
			if err != nil {
				return nil, nil, err
			}
			devices = append(devices, ds...)
		}
	}
	// Building-level devices (meter, electric) read whole-building aggregates.
	for _, a := range cfg.BuildingDevices {
		ds, err := expandArchetype(cfg.NamePrefix, b, ctrl, config.Floor{Name: "building", Title: "Building"}, config.Room{}, nil, a, counters)
		if err != nil {
			return nil, nil, err
		}
		devices = append(devices, ds...)
	}
	return b, devices, nil
}

// expandArchetype generates Count devices for one archetype, registering each
// device's updaters on the building. room is nil for building-level devices.
func expandArchetype(prefix string, b *Building, ctrl *controller, fc config.Floor, rc config.Room, room *Room, a config.Archetype, counters map[string]int) ([]Device, error) {
	var dir string
	if room != nil {
		dir = fmt.Sprintf("%s/%s/%s", prefix, fc.Name, rc.Name)
	} else {
		dir = fmt.Sprintf("%s/building", prefix)
	}
	key := dir + "/" + a.Type
	forceable := archetypes[a.Type].Forceable

	var devices []Device
	for i := 0; i < count(a); i++ {
		counters[key]++
		n := counters[key]
		name := fmt.Sprintf("%s/%s-%02d", dir, a.Type, n)

		features, updaters, subsystem, err := buildArchetype(a, room)
		if err != nil {
			return nil, err
		}
		for _, u := range updaters {
			b.AddUpdater(u)
		}
		// A forceable device (lighting, fcu, occupancy) exposes the MockDeviceApi so
		// its room input can be driven live; the engine then responds via the coupling.
		if room != nil && len(forceable) > 0 {
			ctrl.register(name, room, forceable)
			features = append(features, node.HasServer(mockpb.RegisterMockDeviceApiServer, mockpb.MockDeviceApiServer(ctrl)))
		}
		devices = append(devices, Device{
			Name:     name,
			Metadata: deviceMetadata(name, deviceTitle(a, n), subsystem, fc, rc),
			Features: features,
		})
	}
	return devices, nil
}

// archetypeDesc is the single descriptor for a device archetype: its display
// defaults, the subsystem its devices report under, its scoping rules, and the
// constructor for its trait servers and updaters. The registry below replaces the
// per-concern switch statements (build, room-scoping, occupancy-driven, display
// name) with one table, so adding an archetype is a single entry.
type archetypeDesc struct {
	Type            string
	DisplayName     string
	Subsystem       string
	RoomScoped      bool // reads per-room state, so cannot be a building-level device
	OccupancyDriven bool // readings come entirely from room occupancy
	// Traits is the set of SmartCore traits a device of this archetype exposes. It
	// is the authoritative definition of the archetype's capabilities, expressed in
	// the standard trait vocabulary rather than a driver-private string: the
	// expansion announces exactly these traits (see buildArchetype), and every
	// Forceable input trait must appear here. An archetype is therefore a named
	// device type composed of multiple SmartCore traits — e.g. an fcu is
	// {AirTemperature, FanSpeed, OnOff} — which a config UI can read to
	// present archetypes in terms of the trait standard.
	Traits []trait.Name
	Build  buildFunc
	// Forceable maps the archetype's input traits to the engine override they drive
	// via the MockDeviceApi. Output-only archetypes (derived from the simulation)
	// leave this nil and expose no MockDeviceApi. Its keys are always a subset of Traits.
	Forceable map[trait.Name]roomOverride
}

// buildFunc constructs the trait servers (as node Features) and the per-tick
// updaters for one device of an archetype. The HasTrait features are added by
// buildArchetype from the descriptor's Traits, so Build returns servers only.
// room is nil for building-level devices.
type buildFunc func(a config.Archetype, room *Room) (features []node.Feature, updaters []Updater)

var archetypeList = []archetypeDesc{
	{Type: ArchetypeLighting, DisplayName: "Light", Subsystem: "lighting", RoomScoped: true, Build: buildLighting,
		Traits:    []trait.Name{trait.Light},
		Forceable: map[trait.Name]roomOverride{trait.Light: overrideLight}},
	{Type: ArchetypeFCU, DisplayName: "Fan Coil Unit", Subsystem: "hvac", RoomScoped: true, Build: buildFCU,
		Traits:    []trait.Name{trait.AirTemperature, trait.FanSpeed, trait.OnOff},
		Forceable: map[trait.Name]roomOverride{trait.AirTemperature: overrideSetPoint, trait.FanSpeed: overrideFan}},
	{Type: ArchetypePIR, DisplayName: "Occupancy Sensor", Subsystem: "sensors", RoomScoped: true, OccupancyDriven: true, Build: buildOccupancy,
		Traits:    []trait.Name{trait.OccupancySensor},
		Forceable: map[trait.Name]roomOverride{trait.OccupancySensor: overrideOccupancy}},
	{Type: ArchetypeOccupancy, DisplayName: "Occupancy Sensor", Subsystem: "sensors", RoomScoped: true, OccupancyDriven: true, Build: buildOccupancy,
		Traits:    []trait.Name{trait.OccupancySensor},
		Forceable: map[trait.Name]roomOverride{trait.OccupancySensor: overrideOccupancy}},
	{Type: ArchetypeMotion, DisplayName: "Motion Sensor", Subsystem: "sensors", RoomScoped: true, OccupancyDriven: true, Build: buildMotion,
		Traits: []trait.Name{trait.MotionSensor}},
	{Type: ArchetypeBrightness, DisplayName: "Brightness Sensor", Subsystem: "sensors", RoomScoped: true, Build: buildBrightness,
		Traits: []trait.Name{trait.BrightnessSensor}},
	{Type: ArchetypeAirQuality, DisplayName: "Air Quality Sensor", Subsystem: "sensors", RoomScoped: true, OccupancyDriven: true, Build: buildAirQuality,
		Traits: []trait.Name{trait.AirQualitySensor}},
	{Type: ArchetypeEnterLeave, DisplayName: "Access Reader", Subsystem: "acs", OccupancyDriven: true, Build: buildEnterLeave,
		Traits: []trait.Name{trait.EnterLeaveSensor}},
	{Type: ArchetypeMeter, DisplayName: "Meter", Subsystem: "metering", Build: buildMeter,
		Traits: []trait.Name{meterpb.TraitName}},
	{Type: ArchetypeElectric, DisplayName: "Electricity Meter", Subsystem: "metering", Build: buildElectric,
		Traits: []trait.Name{trait.Electric}},
}

// archetypes indexes archetypeList by Type, panicking on a duplicate so a
// copy-paste slip in the list fails loudly at startup rather than silently
// shadowing an earlier entry.
var archetypes = func() map[string]archetypeDesc {
	m := make(map[string]archetypeDesc, len(archetypeList))
	for _, d := range archetypeList {
		if _, dup := m[d.Type]; dup {
			panic(fmt.Sprintf("sim: duplicate archetype type %q", d.Type))
		}
		// A forceable input must be a trait the archetype actually exposes, so the
		// declared trait set stays the single source of truth for its capabilities.
		declared := make(map[trait.Name]bool, len(d.Traits))
		for _, tn := range d.Traits {
			declared[tn] = true
		}
		for tn := range d.Forceable {
			if !declared[tn] {
				panic(fmt.Sprintf("sim: archetype %q forces trait %q it does not declare in Traits", d.Type, tn))
			}
		}
		m[d.Type] = d
	}
	return m
}()

// buildArchetype constructs the trait servers and updaters for a single device of
// the given archetype. room is nil for building-level aggregates.
func buildArchetype(a config.Archetype, room *Room) (features []node.Feature, updaters []Updater, subsystem string, err error) {
	desc, ok := archetypes[a.Type]
	if !ok {
		return nil, nil, "", fmt.Errorf("unknown archetype type %q", a.Type)
	}
	// Room-scoped archetypes read per-room state; rejecting them here (rather than
	// letting their updater dereference a nil room on the engine goroutine) keeps a
	// misconfigured buildingDevices entry a config error instead of a panic.
	if room == nil && desc.RoomScoped {
		return nil, nil, "", fmt.Errorf("archetype %q is room-scoped and cannot be used as a building-level device", a.Type)
	}
	features, updaters = desc.Build(a, room)
	// Announce the archetype's traits from its descriptor rather than from each
	// Build func, so the registry's Traits is the single source of truth for what
	// the device exposes (Build returns the trait servers only).
	for _, tn := range desc.Traits {
		features = append(features, node.HasTrait(tn))
	}
	return features, updaters, desc.Subsystem, nil
}

// The trait models below are created with resource.WithNoDuplicates so an updater
// that re-publishes an unchanged value each tick produces no Pull emission. This
// keeps steady rooms (overnight, weekends, empty meeting rooms) from churning the
// history automations; values that genuinely change each tick (demand, lux,
// temperature) still emit on every change.

func buildLighting(_ config.Archetype, room *Room) (features []node.Feature, updaters []Updater) {
	model := lightpb.NewModel(
		lightpb.WithPreset(0, &lightpb.LightPreset{Name: "off", Title: "Off"}),
		lightpb.WithPreset(40, &lightpb.LightPreset{Name: "low", Title: "Low"}),
		lightpb.WithPreset(60, &lightpb.LightPreset{Name: "med", Title: "Normal"}),
		lightpb.WithPreset(80, &lightpb.LightPreset{Name: "high", Title: "High"}),
		lightpb.WithPreset(100, &lightpb.LightPreset{Name: "full", Title: "Full"}),
		resource.WithNoDuplicates(),
	)
	server := lightpb.NewModelServer(model)
	features = []node.Feature{
		node.HasServer(lightpb.RegisterLightApiServer, lightpb.LightApiServer(server)),
		node.HasServer(lightpb.RegisterLightInfoServer, lightpb.LightInfoServer(server)),
	}
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		_, _ = model.UpdateBrightness(&lightpb.Brightness{LevelPercent: float32(room.LightLevel)},
			resource.WithUpdatePaths("level_percent"))
	})}
	return features, updaters
}

func buildFCU(_ config.Archetype, room *Room) (features []node.Feature, updaters []Updater) {
	atModel := airtemperaturepb.NewModel(resource.WithNoDuplicates())
	fanModel := fanspeedpb.NewModel(fanspeedpb.WithPresets(fanPresets...), resource.WithNoDuplicates())
	onModel := onoffpb.NewModel(resource.WithInitialValue(&onoffpb.OnOff{State: onoffpb.OnOff_OFF}), resource.WithNoDuplicates())
	features = []node.Feature{
		node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(atModel))),
		node.HasServer(fanspeedpb.RegisterFanSpeedApiServer, fanspeedpb.FanSpeedApiServer(fanspeedpb.NewModelServer(fanModel))),
		node.HasServer(onoffpb.RegisterOnOffApiServer, onoffpb.OnOffApiServer(onoffpb.NewModelServer(onModel))),
	}
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		_, _ = atModel.UpdateAirTemperature(&airtemperaturepb.AirTemperature{
			AmbientTemperature: &typespb.Temperature{ValueCelsius: room.TempC},
			TemperatureGoal: &airtemperaturepb.AirTemperature_TemperatureSetPoint{
				TemperatureSetPoint: &typespb.Temperature{ValueCelsius: room.setPoint()},
			},
		})
		_, _ = fanModel.UpdateFanSpeed(&fanspeedpb.FanSpeed{Percentage: float32(room.FanPct)})
		state := onoffpb.OnOff_OFF
		if room.FanPct > 1 {
			state = onoffpb.OnOff_ON
		}
		_, _ = onModel.UpdateOnOff(&onoffpb.OnOff{State: state})
	})}
	return features, updaters
}

func buildOccupancy(_ config.Archetype, room *Room) (features []node.Feature, updaters []Updater) {
	model := occupancysensorpb.NewModel(resource.WithNoDuplicates())
	features = []node.Feature{
		node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model))),
	}
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		occ := &occupancysensorpb.Occupancy{PeopleCount: int32(room.Occupants)}
		if room.Occupants == 0 {
			occ.State = occupancysensorpb.Occupancy_UNOCCUPIED
		} else {
			occ.State = occupancysensorpb.Occupancy_OCCUPIED
		}
		_, _ = model.SetOccupancy(occ, resource.WithUpdatePaths("state", "people_count"))
	})}
	return features, updaters
}

func buildMotion(_ config.Archetype, room *Room) (features []node.Feature, updaters []Updater) {
	model := motionsensorpb.NewModel(resource.WithNoDuplicates())
	features = []node.Feature{
		node.HasServer(motionsensorpb.RegisterMotionSensorApiServer, motionsensorpb.MotionSensorApiServer(motionsensorpb.NewModelServer(model))),
	}
	// Emit only on a transition. StateChangeTime would otherwise be re-stamped every
	// tick, making each message distinct and defeating the model's deduplication.
	lastState := motionsensorpb.MotionDetection_STATE_UNSPECIFIED
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		state := motionsensorpb.MotionDetection_NOT_DETECTED
		if room.Occupants > 0 {
			state = motionsensorpb.MotionDetection_DETECTED
		}
		if state == lastState {
			return
		}
		lastState = state
		_, _ = model.SetMotionDetection(&motionsensorpb.MotionDetection{State: state, StateChangeTime: timestamppb.New(now)})
	})}
	return features, updaters
}

func buildBrightness(_ config.Archetype, room *Room) (features []node.Feature, updaters []Updater) {
	model := brightnesssensorpb.NewModel(resource.WithNoDuplicates())
	features = []node.Feature{
		node.HasServer(brightnesssensorpb.RegisterBrightnessSensorApiServer, brightnesssensorpb.BrightnessSensorApiServer(brightnesssensorpb.NewModelServer(model))),
	}
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		// Lux = daylight through the windows + the room's own electric lighting.
		// b.day is the daylight factor Tick already computed for this instant.
		lux := b.day*800 + room.LightLevel/100*400
		_, _ = model.UpdateAmbientBrightness(&brightnesssensorpb.AmbientBrightness{BrightnessLux: float32(lux)},
			resource.WithUpdatePaths("brightness_lux"))
	})}
	return features, updaters
}

func buildAirQuality(_ config.Archetype, room *Room) (features []node.Feature, updaters []Updater) {
	model := airqualitysensorpb.NewModel(resource.WithNoDuplicates())
	features = []node.Feature{
		node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(model))),
	}
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		co2 := float32(room.CO2ppm)
		comfort := airqualitysensorpb.AirQuality_COMFORTABLE
		if co2 > 1000 {
			comfort = airqualitysensorpb.AirQuality_UNCOMFORTABLE
		}
		// Score 100 at baseline, falling as CO2 rises.
		score := float32(min(max(100-(room.CO2ppm-420)/12, 0), 100))
		_, _ = model.UpdateAirQuality(&airqualitysensorpb.AirQuality{
			CarbonDioxideLevel: &co2,
			Comfort:            comfort,
			Score:              &score,
		})
	})}
	return features, updaters
}

func buildEnterLeave(_ config.Archetype, room *Room) (features []node.Feature, updaters []Updater) {
	model := enterleavesensorpb.NewModel()
	features = []node.Feature{
		node.HasServer(enterleavesensorpb.RegisterEnterLeaveSensorApiServer, enterleavesensorpb.EnterLeaveSensorApiServer(enterleavesensorpb.NewModelServer(model))),
	}
	// Building-level enter/leave aggregates all rooms; room-level tracks just its room.
	var lastEnter, lastLeave int64
	primed := false
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		enter, leave := enterLeaveTotals(b, room)
		if primed && enter == lastEnter && leave == lastLeave {
			return
		}
		emit := func(dir enterleavesensorpb.EnterLeaveEvent_Direction) {
			// Totals truncate to int32 for the proto fields; sim-scale
			// totals stay far below the limit.
			e32, l32 := int32(enter), int32(leave)
			_ = model.CreateEnterLeaveEvent(&enterleavesensorpb.EnterLeaveEvent{
				Direction:  dir,
				EnterTotal: &e32,
				LeaveTotal: &l32,
			})
		}
		switch {
		case !primed:
			// Baseline event so the sensor reports totals before any movement.
			emit(enterleavesensorpb.EnterLeaveEvent_DIRECTION_UNSPECIFIED)
		default:
			// One event per direction that changed this tick, so simultaneous
			// enters and leaves (e.g. across rooms at building level) are both seen.
			if enter != lastEnter {
				emit(enterleavesensorpb.EnterLeaveEvent_ENTER)
			}
			if leave != lastLeave {
				emit(enterleavesensorpb.EnterLeaveEvent_LEAVE)
			}
		}
		primed = true
		lastEnter, lastLeave = enter, leave
	})}
	return features, updaters
}

func buildMeter(_ config.Archetype, _ *Room) (features []node.Feature, updaters []Updater) {
	model := meterpb.NewModel(resource.WithNoDuplicates())
	info := &meterpb.InfoServer{MeterReading: &meterpb.MeterReadingSupport{
		ResourceSupport: &typespb.ResourceSupport{Readable: true, Observable: true},
		UsageUnit:       "kWh",
	}}
	features = []node.Feature{
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(model))),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(info)),
	}
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		_, _ = model.UpdateMeterReading(&meterpb.MeterReading{
			Usage:     float32(b.MeterKWh),
			StartTime: timestamppb.New(b.meterFrom),
			EndTime:   timestamppb.New(now),
		})
	})}
	return features, updaters
}

func buildElectric(_ config.Archetype, _ *Room) (features []node.Feature, updaters []Updater) {
	model := electricpb.NewModel(resource.WithNoDuplicates())
	features = []node.Feature{
		node.HasServer(electricpb.RegisterElectricApiServer, electricpb.ElectricApiServer(electricpb.NewModelServer(model))),
	}
	const voltage, pf float32 = 240, 0.95
	// Constant for every tick, so box/compute them once per device rather than
	// on the hot path. reactiveFactor = sin(acos(pf)) = sqrt(1 - pf²).
	voltagePtr, pfPtr := ptr(voltage), ptr(pf)
	reactiveFactor := float32(math.Sqrt(float64(1 - pf*pf)))
	updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
		real := float32(b.DemandW)
		apparent := real / pf
		current := apparent / voltage
		reactive := apparent * reactiveFactor
		_, _ = model.UpdateDemand(&electricpb.ElectricDemand{
			Current:       current,
			Voltage:       voltagePtr,
			PowerFactor:   pfPtr,
			RealPower:     ptr(real),
			ApparentPower: ptr(apparent),
			ReactivePower: ptr(reactive),
		})
	})}
	return features, updaters
}

var fanPresets = []fanspeedpb.Preset{
	{Name: "off", Percentage: 0},
	{Name: "low", Percentage: 15},
	{Name: "med", Percentage: 40},
	{Name: "high", Percentage: 75},
	{Name: "full", Percentage: 100},
}

// enterLeaveTotals returns the enter/leave totals for room, or the whole-building
// sum when room is nil (a building-level enterleave device).
func enterLeaveTotals(b *Building, room *Room) (enter, leave int64) {
	if room != nil {
		return room.EnterTotal, room.LeaveTotal
	}
	for _, f := range b.Floors {
		for _, r := range f.Rooms {
			enter += r.EnterTotal
			leave += r.LeaveTotal
		}
	}
	return enter, leave
}

func deviceMetadata(name, title, subsystem string, fc config.Floor, rc config.Room) *metadatapb.Metadata {
	md := &metadatapb.Metadata{
		Name:       name,
		Appearance: &metadatapb.Metadata_Appearance{Title: title},
		Membership: &metadatapb.Metadata_Membership{Subsystem: subsystem},
	}
	floorTitle := fc.Title
	if floorTitle == "" {
		floorTitle = fc.Name
	}
	roomTitle := rc.Title
	if roomTitle == "" {
		roomTitle = rc.Name
	}
	if floorTitle != "" || roomTitle != "" {
		md.Location = &metadatapb.Metadata_Location{Floor: floorTitle, Zone: roomTitle}
	}
	return md
}

// deviceTitle numbers the title with the same per-location counter as the device
// name, so "Light 3" is always lighting-03.
func deviceTitle(a config.Archetype, n int) string {
	base := a.Title
	if base == "" {
		base = displayName(a.Type)
	}
	return fmt.Sprintf("%s %d", base, n)
}

// displayName is the human-readable default for an archetype type, used when a
// device sets no explicit title.
func displayName(typ string) string {
	if d, ok := archetypes[typ]; ok {
		return d.DisplayName
	}
	return typ
}

func count(a config.Archetype) int {
	if a.Count <= 0 {
		return 1
	}
	return a.Count
}

func ratedPower(a config.Archetype, def float64) float64 {
	if a.RatedPowerW > 0 {
		return a.RatedPowerW
	}
	return def
}

// occupancyDriven reports whether an archetype's readings come entirely from room
// occupancy, making it permanently empty/idle in a room whose max occupancy
// resolves to zero. Lighting, brightness and fcu are excluded: they still produce
// sensible time-of-day output in an unoccupied room (e.g. corridor lighting).
func occupancyDriven(typ string) bool {
	return archetypes[typ].OccupancyDriven
}

// zeroOccupancyRooms returns the floor/room paths of rooms that contain
// occupancy-driven archetypes but resolve to a max occupancy of zero — almost
// always a forgotten maxOccupancy rather than intent.
func zeroOccupancyRooms(cfg config.Root) []string {
	var out []string
	for _, fc := range cfg.Floors {
		roomMax := sharedOccupancy(fc)
		for _, rc := range fc.Rooms {
			if roomMax(rc) > 0 {
				continue
			}
			for _, a := range rc.Archetypes {
				if occupancyDriven(a.Type) {
					out = append(out, fc.Name+"/"+rc.Name)
					break
				}
			}
		}
	}
	return out
}

// sharedOccupancy returns a function giving each room its max occupancy: the room's
// own value if set, otherwise the floor's occupancy shared evenly between rooms that
// don't set their own.
func sharedOccupancy(fc config.Floor) func(config.Room) int {
	explicit, rooms := 0, 0
	for _, r := range fc.Rooms {
		if r.MaxOccupancy > 0 {
			explicit += r.MaxOccupancy
		} else {
			rooms++
		}
	}
	share := 0
	if rooms > 0 {
		remaining := fc.MaxOccupancy - explicit
		if remaining < 0 {
			remaining = 0
		}
		share = remaining / rooms
	}
	return func(r config.Room) int {
		if r.MaxOccupancy > 0 {
			return r.MaxOccupancy
		}
		return share
	}
}
