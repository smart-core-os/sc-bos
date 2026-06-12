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
	"github.com/smart-core-os/sc-bos/pkg/proto/motionsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
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

	// Device numbering is keyed by location+type rather than per archetype entry,
	// so a room listing the same type twice (e.g. to mix titles or rated powers)
	// continues lighting-01, -02, -03 instead of generating colliding names.
	counters := make(map[string]int)
	var devices []Device
	for _, rb := range built {
		for _, a := range rb.rc.Archetypes {
			ds, err := expandArchetype(cfg.NamePrefix, b, rb.fc, rb.rc, rb.room, a, counters)
			if err != nil {
				return nil, nil, err
			}
			devices = append(devices, ds...)
		}
	}
	// Building-level devices (meter, electric) read whole-building aggregates.
	for _, a := range cfg.BuildingDevices {
		ds, err := expandArchetype(cfg.NamePrefix, b, config.Floor{Name: "building", Title: "Building"}, config.Room{}, nil, a, counters)
		if err != nil {
			return nil, nil, err
		}
		devices = append(devices, ds...)
	}
	return b, devices, nil
}

// expandArchetype generates Count devices for one archetype, registering each
// device's updaters on the building. room is nil for building-level devices.
func expandArchetype(prefix string, b *Building, fc config.Floor, rc config.Room, room *Room, a config.Archetype, counters map[string]int) ([]Device, error) {
	var dir string
	if room != nil {
		dir = fmt.Sprintf("%s/%s/%s", prefix, fc.Name, rc.Name)
	} else {
		dir = fmt.Sprintf("%s/building", prefix)
	}
	key := dir + "/" + a.Type

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
		devices = append(devices, Device{
			Name:     name,
			Metadata: deviceMetadata(name, deviceTitle(a, n), subsystem, fc, rc),
			Features: features,
		})
	}
	return devices, nil
}

// buildArchetype constructs the trait servers and updaters for a single device of
// the given archetype. room is nil for building-level aggregates.
func buildArchetype(a config.Archetype, room *Room) (features []node.Feature, updaters []Updater, subsystem string, err error) {
	// Room-scoped archetypes read per-room state; rejecting them here (rather than
	// letting their updater dereference a nil room on the engine goroutine) keeps a
	// misconfigured buildingDevices entry a config error instead of a panic.
	if room == nil && roomScoped(a.Type) {
		return nil, nil, "", fmt.Errorf("archetype %q is room-scoped and cannot be used as a building-level device", a.Type)
	}
	switch a.Type {
	case ArchetypeLighting:
		model := lightpb.NewModel(
			lightpb.WithPreset(0, &lightpb.LightPreset{Name: "off", Title: "Off"}),
			lightpb.WithPreset(40, &lightpb.LightPreset{Name: "low", Title: "Low"}),
			lightpb.WithPreset(60, &lightpb.LightPreset{Name: "med", Title: "Normal"}),
			lightpb.WithPreset(80, &lightpb.LightPreset{Name: "high", Title: "High"}),
			lightpb.WithPreset(100, &lightpb.LightPreset{Name: "full", Title: "Full"}),
		)
		server := lightpb.NewModelServer(model)
		features = []node.Feature{
			node.HasServer(lightpb.RegisterLightApiServer, lightpb.LightApiServer(server)),
			node.HasServer(lightpb.RegisterLightInfoServer, lightpb.LightInfoServer(server)),
			node.HasTrait(trait.Light),
		}
		updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
			_, _ = model.UpdateBrightness(&lightpb.Brightness{LevelPercent: float32(room.LightLevel)},
				resource.WithUpdatePaths("level_percent"))
		})}
		features = append(features, statusFeatures()...)
		return features, updaters, "lighting", nil

	case ArchetypeFCU:
		atModel := airtemperaturepb.NewModel()
		fanModel := fanspeedpb.NewModel(fanspeedpb.WithPresets(fanPresets...))
		onModel := onoffpb.NewModel(resource.WithInitialValue(&onoffpb.OnOff{State: onoffpb.OnOff_OFF}))
		features = []node.Feature{
			node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(atModel))),
			node.HasTrait(trait.AirTemperature),
			node.HasServer(fanspeedpb.RegisterFanSpeedApiServer, fanspeedpb.FanSpeedApiServer(fanspeedpb.NewModelServer(fanModel))),
			node.HasTrait(trait.FanSpeed),
			node.HasServer(onoffpb.RegisterOnOffApiServer, onoffpb.OnOffApiServer(onoffpb.NewModelServer(onModel))),
			node.HasTrait(trait.OnOff),
		}
		updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
			_, _ = atModel.UpdateAirTemperature(&airtemperaturepb.AirTemperature{
				AmbientTemperature: &typespb.Temperature{ValueCelsius: room.TempC},
				TemperatureGoal: &airtemperaturepb.AirTemperature_TemperatureSetPoint{
					TemperatureSetPoint: &typespb.Temperature{ValueCelsius: room.SetPointC},
				},
			})
			_, _ = fanModel.UpdateFanSpeed(&fanspeedpb.FanSpeed{Percentage: float32(room.FanPct)})
			state := onoffpb.OnOff_OFF
			if room.FanPct > 1 {
				state = onoffpb.OnOff_ON
			}
			_, _ = onModel.UpdateOnOff(&onoffpb.OnOff{State: state})
		})}
		features = append(features, statusFeatures()...)
		return features, updaters, "hvac", nil

	case ArchetypePIR, ArchetypeOccupancy:
		model := occupancysensorpb.NewModel()
		features = []node.Feature{
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model))),
			node.HasTrait(trait.OccupancySensor),
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
		return features, updaters, "sensors", nil

	case ArchetypeMotion:
		model := motionsensorpb.NewModel()
		features = []node.Feature{
			node.HasServer(motionsensorpb.RegisterMotionSensorApiServer, motionsensorpb.MotionSensorApiServer(motionsensorpb.NewModelServer(model))),
			node.HasTrait(trait.MotionSensor),
		}
		updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
			state := motionsensorpb.MotionDetection_NOT_DETECTED
			if room.Occupants > 0 {
				state = motionsensorpb.MotionDetection_DETECTED
			}
			_, _ = model.SetMotionDetection(&motionsensorpb.MotionDetection{State: state, StateChangeTime: timestamppb.New(now)})
		})}
		return features, updaters, "sensors", nil

	case ArchetypeBrightness:
		model := brightnesssensorpb.NewModel()
		features = []node.Feature{
			node.HasServer(brightnesssensorpb.RegisterBrightnessSensorApiServer, brightnesssensorpb.BrightnessSensorApiServer(brightnesssensorpb.NewModelServer(model))),
			node.HasTrait(trait.BrightnessSensor),
		}
		updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
			// Lux = daylight through the windows + the room's own electric lighting.
			// b.day is the daylight factor Tick already computed for this instant.
			lux := b.day*800 + room.LightLevel/100*400
			_, _ = model.UpdateAmbientBrightness(&brightnesssensorpb.AmbientBrightness{BrightnessLux: float32(lux)},
				resource.WithUpdatePaths("brightness_lux"))
		})}
		return features, updaters, "sensors", nil

	case ArchetypeAirQuality:
		model := airqualitysensorpb.NewModel()
		features = []node.Feature{
			node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(model))),
			node.HasTrait(trait.AirQualitySensor),
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
		return features, updaters, "sensors", nil

	case ArchetypeEnterLeave:
		model := enterleavesensorpb.NewModel()
		features = []node.Feature{
			node.HasServer(enterleavesensorpb.RegisterEnterLeaveSensorApiServer, enterleavesensorpb.EnterLeaveSensorApiServer(enterleavesensorpb.NewModelServer(model))),
			node.HasTrait(trait.EnterLeaveSensor),
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
		return features, updaters, "acs", nil

	case ArchetypeMeter:
		model := meterpb.NewModel()
		info := &meterpb.InfoServer{MeterReading: &meterpb.MeterReadingSupport{
			ResourceSupport: &typespb.ResourceSupport{Readable: true, Observable: true},
			UsageUnit:       "kWh",
		}}
		features = []node.Feature{
			node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(model))),
			node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(info)),
			node.HasTrait(meterpb.TraitName),
		}
		updaters = []Updater{updaterFunc(func(now time.Time, b *Building) {
			_, _ = model.UpdateMeterReading(&meterpb.MeterReading{
				Usage:     float32(b.MeterKWh),
				StartTime: timestamppb.New(b.meterFrom),
				EndTime:   timestamppb.New(now),
			})
		})}
		return features, updaters, "metering", nil

	case ArchetypeElectric:
		model := electricpb.NewModel()
		features = []node.Feature{
			node.HasServer(electricpb.RegisterElectricApiServer, electricpb.ElectricApiServer(electricpb.NewModelServer(model))),
			node.HasTrait(trait.Electric),
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
		return features, updaters, "metering", nil
	}
	return nil, nil, "", fmt.Errorf("unknown archetype type %q", a.Type)
}

var fanPresets = []fanspeedpb.Preset{
	{Name: "off", Percentage: 0},
	{Name: "low", Percentage: 15},
	{Name: "med", Percentage: 40},
	{Name: "high", Percentage: 75},
	{Name: "full", Percentage: 100},
}

// statusFeatures adds a Status trait reporting a static NOMINAL state. The
// simulation does not vary per-device health (driver-level health does), so it
// registers no updater.
func statusFeatures() []node.Feature {
	model := statuspb.NewModel()
	_, _ = model.UpdateProblem(&statuspb.StatusLog_Problem{Level: statuspb.StatusLog_NOMINAL, Description: "All systems operational"})
	return []node.Feature{
		node.HasServer(statuspb.RegisterStatusApiServer, statuspb.StatusApiServer(statuspb.NewModelServer(model))),
		node.HasTrait(statuspb.TraitName),
	}
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

func displayName(typ string) string {
	switch typ {
	case ArchetypeLighting:
		return "Light"
	case ArchetypeFCU:
		return "Fan Coil Unit"
	case ArchetypePIR, ArchetypeOccupancy:
		return "Occupancy Sensor"
	case ArchetypeMotion:
		return "Motion Sensor"
	case ArchetypeBrightness:
		return "Brightness Sensor"
	case ArchetypeAirQuality:
		return "Air Quality Sensor"
	case ArchetypeMeter:
		return "Meter"
	case ArchetypeElectric:
		return "Electricity Meter"
	case ArchetypeEnterLeave:
		return "Access Reader"
	}
	return typ
}

func count(a config.Archetype) int {
	if a.Count <= 0 {
		return 1
	}
	return a.Count
}

// roomScoped reports whether an archetype belongs to a specific room and reads
// per-room state. These cannot be configured as building-level devices (where
// room is nil); only meter, electric and enterleave aggregate at the building.
func roomScoped(typ string) bool {
	switch typ {
	case ArchetypeLighting, ArchetypeFCU, ArchetypePIR, ArchetypeOccupancy,
		ArchetypeMotion, ArchetypeBrightness, ArchetypeAirQuality:
		return true
	}
	return false
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
	switch typ {
	case ArchetypePIR, ArchetypeOccupancy, ArchetypeMotion,
		ArchetypeAirQuality, ArchetypeEnterLeave:
		return true
	}
	return false
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
