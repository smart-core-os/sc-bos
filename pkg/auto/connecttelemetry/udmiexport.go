package connecttelemetry

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
)

// UDMI event subfolders, the trailing topic segment Connect classifies on.
const (
	subfolderPointset  = "pointset"
	subfolderDiscovery = "discovery"
)

// eventTopic builds the UDMI event topic for a device under the Connect ingest
// grammar: <prefix>/devices/<deviceRef>/events/<subfolder>.
//
// NB deviceRef is currently the raw Smart Core device name, which contains '/'.
// That collides with topic segmentation and is a known, deliberately-parked gap
// (see docs/connect-telemetry-ingest.md) — do not rely on the segment count here.
func eventTopic(prefix, deviceRef, subfolder string) string {
	return strings.TrimRight(prefix, "/") + "/devices/" + deviceRef + "/events/" + subfolder
}

func pointsetTopic(prefix, deviceRef string) string {
	return eventTopic(prefix, deviceRef, subfolderPointset)
}

func discoveryTopic(prefix, deviceRef string) string {
	return eventTopic(prefix, deviceRef, subfolderDiscovery)
}

// meterTelemetry maps a meter reading to a UDMI pointset event. The point key is
// chosen by naming: DBO standard field names (energy_accumulator, …) make the
// building-config translation an identity mapping; raw names (usage/produced) leave
// the name→field mapping to the consumer.
//
// The reading instant is MeterReading.end_time, falling back to now when unset.
func meterTelemetry(r *meterpb.MeterReading, support *meterpb.MeterReadingSupport, now time.Time, naming dbo.Naming) udmi.PointsetEvent {
	ts := now
	if r.GetEndTime() != nil {
		ts = r.GetEndTime().AsTime()
	}
	points := udmi.PointsEvent{}
	// Meter readings are float32 (the Meter trait's type). For a cumulative
	// accumulator that is a resolution ceiling once the running total grows large
	// (~7 significant digits; reached sooner for meters reporting in Wh) — a known,
	// upstream-bound limitation tracked in docs/connect-telemetry-ingest.md.
	// Deliberately kept as float32 rather than widened to float64: encoding/json
	// emits the shortest decimal that round-trips the float32, whereas a float64
	// conversion would surface the float32's binary artifacts on the wire.
	for _, f := range dbo.MeterFields(support.GetUsageUnit(), support.GetProducedUnit()) {
		value := r.GetUsage()
		if f.Exported {
			value = r.GetProduced()
		}
		points[f.PointName(naming)] = udmi.PointValue{PresentValue: value}
	}
	return udmi.PointsetEvent{Timestamp: ts, Version: udmi.PointsetVersion, Points: points}
}

// meterInventory contributes the meter's declared points to discovery, keyed by the
// same name mode as the telemetry (see meterTelemetry). Meters are read-only, so no
// point is marked writable. Units carry the raw device unit string.
func meterInventory(support *meterpb.MeterReadingSupport, naming dbo.Naming) map[string]udmi.MetadataPoint {
	inv := map[string]udmi.MetadataPoint{}
	for _, f := range dbo.MeterFields(support.GetUsageUnit(), support.GetProducedUnit()) {
		inv[f.PointName(naming)] = udmi.MetadataPoint{Units: f.RawUnit}
	}
	return inv
}

// Units declared in discovery for ResourceUse points. ResourceUse leaves them
// implicit: utilisation is a 0-100 percentage and usage/limit are byte counts.
const (
	unitPercent = "percent"
	unitBytes   = "bytes"
)

// resourceUseTelemetry maps a node's ResourceUse to UDMI points. The names are
// DBO-style fields from Connect's catalogue, which extends DBO with the `cpu`,
// `memory` and `disk` components and the `datasize` measurement (SCC-978); they
// are the same in both naming modes. Unset optional fields produce no point.
// Disks are keyed by mount point (see diskInstance), so their list order doesn't
// matter.
//
// ResourceUse carries no timestamp; the caller stamps the reading with the time
// the node last sampled it (the model's change time).
func resourceUseTelemetry(r *resourceusepb.ResourceUse) udmi.PointsEvent {
	points := udmi.PointsEvent{}
	forEachResourceUsePoint(r, func(name string, value any, _ string, _ string) {
		points[name] = udmi.PointValue{PresentValue: value}
	})
	return points
}

// resourceUseInventory declares the points resourceUseTelemetry emits for r, with
// their units. Disk points carry the raw mount point as the model-point ref.
func resourceUseInventory(r *resourceusepb.ResourceUse) map[string]udmi.MetadataPoint {
	inv := map[string]udmi.MetadataPoint{}
	forEachResourceUsePoint(r, func(name string, _ any, units, ref string) {
		inv[name] = udmi.MetadataPoint{Units: units, Ref: ref}
	})
	return inv
}

// forEachResourceUsePoint calls fn for every set ResourceUse field we export, so
// telemetry and discovery can't drift apart. ref is the disk mount point, or empty.
func forEachResourceUsePoint(r *resourceusepb.ResourceUse, fn func(name string, value any, units, ref string)) {
	if cpu := r.GetCpu(); cpu != nil && cpu.Utilization != nil {
		fn("cpu_percentage_sensor", cpu.GetUtilization(), unitPercent, "")
	}
	if mem := r.GetMemory(); mem != nil {
		if mem.Utilization != nil {
			fn("memory_percentage_sensor", mem.GetUtilization(), unitPercent, "")
		}
		if mem.Usage != nil {
			fn("memory_datasize_sensor", mem.GetUsage(), unitBytes, "")
		}
		if mem.Limit != nil {
			fn("memory_datasize_capacity", mem.GetLimit(), unitBytes, "")
		}
	}
	for _, d := range r.GetDisks() {
		mount := d.GetMountPoint()
		n := diskInstance(mount)
		if d.Utilization != nil {
			fn(fmt.Sprintf("disk_percentage_sensor_%d", n), d.GetUtilization(), unitPercent, mount)
		}
		if d.Usage != nil {
			fn(fmt.Sprintf("disk_datasize_sensor_%d", n), d.GetUsage(), unitBytes, mount)
		}
		if d.Limit != nil {
			fn(fmt.Sprintf("disk_datasize_capacity_%d", n), d.GetLimit(), unitBytes, mount)
		}
	}
}

// envPoint is one exported environmental sensor point: its DBO field, the raw
// Smart Core name (the trait's proto field) it takes under dbo.NamingRaw, and the
// unit the trait defines it in, empty for a unitless point.
type envPoint struct {
	field, raw, units string
}

func (p envPoint) name(n dbo.Naming) string {
	if n == dbo.NamingRaw {
		return p.raw
	}
	return p.field
}

// The environmental points, on the fields Connect's BOS CSV import lands the same
// readings on. Their units are fixed by the trait protos, which carry none on the
// wire, so discovery declares them for Connect to convert by.
var (
	pointAirTemperature = envPoint{dbo.FieldZoneAirTemperatureSensor, "ambient_temperature", dbo.UnitDegreesCelsius}
	pointAirHumidity    = envPoint{dbo.FieldZoneAirRelativeHumiditySensor, "ambient_humidity", dbo.UnitPercentRelativeHumidity}
	pointCO2            = envPoint{dbo.FieldZoneAirCO2ConcentrationSensor, "carbon_dioxide_level", dbo.UnitPartsPerMillion}
	pointVOC            = envPoint{dbo.FieldZoneAirVOCConcentrationSensor, "volatile_organic_compounds", dbo.UnitPartsPerMillion}
	pointAirPressure    = envPoint{dbo.FieldAirPressureSensor, "air_pressure", dbo.UnitHectopascals}
	pointPM1            = envPoint{dbo.FieldZoneAirPM1pt0DensitySensor, "particulate_matter_1", dbo.UnitMicrogramsPerCubicMeter}
	pointPM25           = envPoint{dbo.FieldZoneAirPM2pt5DensitySensor, "particulate_matter_25", dbo.UnitMicrogramsPerCubicMeter}
	pointPM10           = envPoint{dbo.FieldZoneAirPM10pt0DensitySensor, "particulate_matter_10", dbo.UnitMicrogramsPerCubicMeter}
	// VOC by mass is its own point, never folded into pointVOC: ppm and mass don't
	// convert without a reference gas.
	pointVOCDensity     = envPoint{dbo.FieldZoneAirVOCDensitySensor, "volatile_organic_compounds_density", dbo.UnitMicrogramsPerCubicMeter}
	pointOccupancyState = envPoint{dbo.FieldZoneOccupancyStatus, "state", ""}
	pointPeopleCount    = envPoint{dbo.FieldZoneOccupancyCount, "people_count", ""}
	pointTemperature    = envPoint{dbo.FieldTemperatureSensor, "measured", dbo.UnitDegreesCelsius}
)

// envPointFunc receives one point of a reading: its name under the naming mode,
// its value and its unit.
type envPointFunc func(name string, value any, units string)

// envTelemetry maps reading r to UDMI points, using forEach, the trait's
// forEach…Point. Building telemetry and discovery from the one callback means
// they can't drift apart.
func envTelemetry[T any](r T, n dbo.Naming, forEach func(T, dbo.Naming, envPointFunc)) udmi.PointsEvent {
	points := udmi.PointsEvent{}
	forEach(r, n, func(name string, value any, _ string) {
		points[name] = udmi.PointValue{PresentValue: value}
	})
	return points
}

// envInventory declares the points envTelemetry emits for r, with their units.
func envInventory[T any](r T, n dbo.Naming, forEach func(T, dbo.Naming, envPointFunc)) map[string]udmi.MetadataPoint {
	inv := map[string]udmi.MetadataPoint{}
	forEach(r, n, func(name string, _ any, units string) {
		inv[name] = udmi.MetadataPoint{Units: units}
	})
	return inv
}

// forEachAirTemperaturePoint calls fn for every set AirTemperature reading we
// export: the ambient temperature and humidity. Set points and mode are
// writable or enum-valued, not sensor readings, so they aren't exported.
func forEachAirTemperaturePoint(r *airtemperaturepb.AirTemperature, n dbo.Naming, fn envPointFunc) {
	if r == nil {
		return
	}
	emit := func(p envPoint, v any) { fn(p.name(n), v, p.units) }
	if t := r.GetAmbientTemperature(); t != nil {
		emit(pointAirTemperature, t.GetValueCelsius())
	}
	if r.AmbientHumidity != nil {
		emit(pointAirHumidity, r.GetAmbientHumidity())
	}
}

// forEachAirQualityPoint calls fn for every set AirQuality reading that has a DBO
// field. Comfort, score and infection risk are vendor-defined indices, and air
// changes per hour has no DBO quantity, so they aren't exported.
func forEachAirQualityPoint(r *airqualitysensorpb.AirQuality, n dbo.Naming, fn envPointFunc) {
	if r == nil {
		return
	}
	emit := func(p envPoint, v *float32) {
		if v != nil {
			fn(p.name(n), *v, p.units)
		}
	}
	emit(pointCO2, r.CarbonDioxideLevel)
	emit(pointVOC, r.VolatileOrganicCompounds)
	emit(pointVOCDensity, r.VolatileOrganicCompoundsDensity)
	emit(pointAirPressure, r.AirPressure)
	emit(pointPM1, r.ParticulateMatter_1)
	emit(pointPM25, r.ParticulateMatter_25)
	emit(pointPM10, r.ParticulateMatter_10)
}

// forEachTemperaturePoint calls fn for a Temperature's measured reading. The set point is
// writable, not a reading, so it isn't exported.
func forEachTemperaturePoint(r *temperaturepb.Temperature, n dbo.Naming, fn envPointFunc) {
	forEachTemperaturePointAs(r, pointTemperature.name(n), fn)
}

// forEachTemperaturePointFor gives the shape sensorCollector takes for a grouped child's
// Temperature, sending its measured reading as point key, see pointKey, in either naming.
func forEachTemperaturePointFor(key string) func(*temperaturepb.Temperature, dbo.Naming, envPointFunc) {
	return func(r *temperaturepb.Temperature, _ dbo.Naming, fn envPointFunc) {
		forEachTemperaturePointAs(r, key, fn)
	}
}

func forEachTemperaturePointAs(r *temperaturepb.Temperature, name string, fn envPointFunc) {
	if t := r.GetMeasured(); t != nil {
		fn(name, t.GetValueCelsius(), pointTemperature.units)
	}
}

// occupancyStates maps OccupancySensor states onto DBO zone_occupancy_status, as
// Connect's BOS CSV import does: IDLE, "likely occupied but no activity", is DBO's
// inactive, and STATE_UNSPECIFIED, no signal either way, is DBO's unknown.
var occupancyStates = map[occupancysensorpb.Occupancy_State]string{
	occupancysensorpb.Occupancy_OCCUPIED:          dbo.OccupancyOccupied,
	occupancysensorpb.Occupancy_UNOCCUPIED:        dbo.OccupancyUnoccupied,
	occupancysensorpb.Occupancy_IDLE:              dbo.OccupancyInactive,
	occupancysensorpb.Occupancy_STATE_UNSPECIFIED: dbo.OccupancyUnknown,
}

// peopleCounting is what an occupancy sensor says about its people count, read
// from OccupancySupport.max_people.
type peopleCounting int

const (
	// countingUndescribed is a sensor whose driver doesn't describe it, as no
	// driver does yet. It may be a counter or a presence detector reporting 0.
	countingUndescribed peopleCounting = iota
	// countingNone is a sensor that says it reports no people count (max_people 0).
	countingNone
	// countingCounts is a sensor that says it counts people (max_people above 0).
	countingCounts
)

// peopleCountingOf reads a sensor's OccupancySupport; nil is an undescribed sensor.
func peopleCountingOf(support *occupancysensorpb.OccupancySupport) peopleCounting {
	switch {
	case support == nil:
		return countingUndescribed
	case support.GetMaxPeople() > 0:
		return countingCounts
	default:
		return countingNone
	}
}

// forEachOccupancyPoint calls fn for an Occupancy's state and headcount. The state
// goes as its canonical DBO state, a string, since Connect holds a number on a
// multistate field; raw naming sends the trait's own state name instead. A state
// this table doesn't know sends no state.
//
// The trait reports a people count of 0 from a sensor that can't count, so a 0
// can be no reading. Which counts go depends on what the sensor says it does:
//   - one that counts sends its count, except a 0 beside an occupied or idle
//     state, which Connect's import reads as no reading too;
//   - one that doesn't count sends none;
//   - an undescribed one sends only a count above 0, so a presence detector such
//     as a lighting PIR never claims an empty room holds 0 people. A counter
//     loses its true zeros until its driver describes it.
func forEachOccupancyPoint(r *occupancysensorpb.Occupancy, counting peopleCounting, n dbo.Naming, fn envPointFunc) {
	if r == nil {
		return
	}
	state := r.GetState()
	if n == dbo.NamingRaw {
		fn(pointOccupancyState.name(n), state.String(), pointOccupancyState.units)
	} else if s, ok := occupancyStates[state]; ok {
		fn(pointOccupancyState.name(n), s, pointOccupancyState.units)
	}
	count := r.GetPeopleCount()
	switch counting {
	case countingNone:
		return
	case countingUndescribed:
		if count == 0 {
			return
		}
	case countingCounts:
		if count == 0 && (state == occupancysensorpb.Occupancy_OCCUPIED || state == occupancysensorpb.Occupancy_IDLE) {
			return
		}
	}
	fn(pointPeopleCount.name(n), count, pointPeopleCount.units)
}

// forEachOccupancyPointFor binds a sensor's counting to forEachOccupancyPoint,
// giving the shape sensorCollector takes.
func forEachOccupancyPointFor(counting peopleCounting) func(*occupancysensorpb.Occupancy, dbo.Naming, envPointFunc) {
	return func(r *occupancysensorpb.Occupancy, n dbo.Naming, fn envPointFunc) {
		forEachOccupancyPoint(r, counting, n, fn)
	}
}

// diskInstance is a disk's DBO numeric instance suffix: the fnv32a hash of its raw
// mount point, so "/" is 705468254. A field name's tokens must all be catalogue
// subfields, so a mount can't be spelt into the name, and a list index would move
// one disk's history onto another when a mount appears or goes. The hash keeps
// each mount on its own series in any order; discovery's ref names the mount.
func diskInstance(mount string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(mount))
	return h.Sum32()
}

// discoveryEvent builds the UDMI device-metadata (discovery) message from the
// device's Smart Core metadata and the declared point inventory.
//
// system.name prefers the appearance title (falling back to the device name),
// tags carry the building subsystem selector, and location carries the floor.
// site is intentionally left empty here: on the Connect path org/site identity is
// supplied by broker enrichment, not by the payload.
func discoveryEvent(meta *metadatapb.Metadata, inventory map[string]udmi.MetadataPoint, now time.Time) udmi.MetadataEvent {
	ev := udmi.MetadataEvent{
		Timestamp: now,
		Version:   udmi.PointsetVersion,
	}
	if meta != nil {
		if a := meta.GetAppearance(); a != nil {
			ev.System.Name = a.GetTitle()
			ev.System.Description = a.GetDescription()
		}
		if ev.System.Name == "" {
			ev.System.Name = meta.GetName()
		}
		if sub := meta.GetMembership().GetSubsystem(); sub != "" {
			ev.System.Tags = append(ev.System.Tags, sub)
		}
		if loc := meta.GetLocation(); loc != nil && loc.GetFloor() != "" {
			ev.System.Location = &udmi.MetadataLocation{Floor: loc.GetFloor()}
		}
	}
	if len(inventory) > 0 {
		ev.Pointset = &udmi.MetadataPointset{Points: inventory}
	}
	return ev
}
