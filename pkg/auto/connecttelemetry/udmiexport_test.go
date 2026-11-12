package connecttelemetry

import (
	"encoding/json"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

func TestEventTopics(t *testing.T) {
	const ref = "van/uk/brum/ugs/meters/elec-main"

	assert.Equal(t,
		"tlm/devices/van/uk/brum/ugs/meters/elec-main/events/pointset",
		pointsetTopic("tlm", ref))
	assert.Equal(t,
		"tlm/devices/van/uk/brum/ugs/meters/elec-main/events/discovery",
		discoveryTopic("tlm", ref))

	// a trailing slash on the prefix must not double up.
	assert.Equal(t,
		"tlm/devices/m1/events/pointset",
		pointsetTopic("tlm/", "m1"))
}

func TestMeterTelemetry(t *testing.T) {
	end := time.Date(2026, 6, 22, 10, 15, 0, 0, time.UTC)
	now := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("usage and produced map to DBO energy fields", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 123.45, Produced: 67.89, EndTime: timestamppb.New(end)}
		support := &meterpb.MeterReadingSupport{UsageUnit: "kWh", ProducedUnit: "kWh"}

		ev := meterTelemetry(r, support, now, dbo.NamingDBO)

		assert.Equal(t, end, ev.Timestamp, "timestamp should be the reading end_time")
		assert.Equal(t, udmi.PointsetVersion, ev.Version)
		require.Contains(t, ev.Points, dbo.FieldEnergyAccumulator)
		require.Contains(t, ev.Points, dbo.FieldExportedEnergyAccumulator)
		assert.Equal(t, float32(123.45), ev.Points[dbo.FieldEnergyAccumulator].PresentValue)
		assert.Equal(t, float32(67.89), ev.Points[dbo.FieldExportedEnergyAccumulator].PresentValue)
	})

	t.Run("falls back to now when end_time unset", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 1}
		ev := meterTelemetry(r, &meterpb.MeterReadingSupport{UsageUnit: "kWh"}, now, dbo.NamingDBO)
		assert.Equal(t, now, ev.Timestamp)
	})

	t.Run("consumption-only meter omits exported when support has no producedUnit", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 10, Produced: 0, EndTime: timestamppb.New(end)}
		support := &meterpb.MeterReadingSupport{UsageUnit: "kWh"} // no ProducedUnit

		ev := meterTelemetry(r, support, now, dbo.NamingDBO)

		require.Contains(t, ev.Points, dbo.FieldEnergyAccumulator)
		assert.NotContains(t, ev.Points, dbo.FieldExportedEnergyAccumulator, "no producedUnit ⇒ no exported series")
	})

	t.Run("without support, only the energy accumulator is emitted", func(t *testing.T) {
		ev := meterTelemetry(&meterpb.MeterReading{Usage: 10, Produced: 5}, nil, now, dbo.NamingDBO)
		require.Contains(t, ev.Points, dbo.FieldEnergyAccumulator)
		assert.NotContains(t, ev.Points, dbo.FieldExportedEnergyAccumulator,
			"without support (no declared producedUnit) exported energy is not claimed")
	})

	t.Run("water meter emits water_volume_accumulator from usage", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 42, EndTime: timestamppb.New(end)}
		ev := meterTelemetry(r, &meterpb.MeterReadingSupport{UsageUnit: "m³"}, now, dbo.NamingDBO)
		require.Contains(t, ev.Points, dbo.FieldWaterVolumeAccumulator)
		assert.NotContains(t, ev.Points, dbo.FieldEnergyAccumulator, "a water meter is not energy")
		assert.Equal(t, float32(42), ev.Points[dbo.FieldWaterVolumeAccumulator].PresentValue)
	})

	t.Run("raw naming emits raw Smart Core point names", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 123.45, Produced: 67.89, EndTime: timestamppb.New(end)}
		support := &meterpb.MeterReadingSupport{UsageUnit: "kWh", ProducedUnit: "kWh"}

		ev := meterTelemetry(r, support, now, dbo.NamingRaw)

		require.Contains(t, ev.Points, dbo.RawPointUsage)
		require.Contains(t, ev.Points, dbo.RawPointProduced)
		assert.NotContains(t, ev.Points, dbo.FieldEnergyAccumulator, "raw mode must not emit DBO names")
		assert.Equal(t, float32(123.45), ev.Points[dbo.RawPointUsage].PresentValue)
		assert.Equal(t, float32(67.89), ev.Points[dbo.RawPointProduced].PresentValue)
	})
}

func TestMeterInventory(t *testing.T) {
	t.Run("usage and produced units (raw strings)", func(t *testing.T) {
		inv := meterInventory(&meterpb.MeterReadingSupport{UsageUnit: "kWh", ProducedUnit: "kWh"}, dbo.NamingDBO)
		require.Contains(t, inv, dbo.FieldEnergyAccumulator)
		require.Contains(t, inv, dbo.FieldExportedEnergyAccumulator)
		// discovery carries the raw device unit string; DBO unit-name mapping is a
		// building-config concern.
		assert.Equal(t, "kWh", inv[dbo.FieldEnergyAccumulator].Units)
		assert.Equal(t, "kWh", inv[dbo.FieldExportedEnergyAccumulator].Units)
		// meters are read-only: nothing writable.
		assert.False(t, inv[dbo.FieldEnergyAccumulator].Writable)
		assert.False(t, inv[dbo.FieldExportedEnergyAccumulator].Writable)
	})

	t.Run("energy accumulator only when no producedUnit", func(t *testing.T) {
		inv := meterInventory(&meterpb.MeterReadingSupport{UsageUnit: "kWh"}, dbo.NamingDBO)
		require.Contains(t, inv, dbo.FieldEnergyAccumulator)
		assert.NotContains(t, inv, dbo.FieldExportedEnergyAccumulator)
	})

	t.Run("nil support still lists the energy accumulator", func(t *testing.T) {
		inv := meterInventory(nil, dbo.NamingDBO)
		require.Contains(t, inv, dbo.FieldEnergyAccumulator)
		assert.Empty(t, inv[dbo.FieldEnergyAccumulator].Units)
	})

	t.Run("water meter lists water_volume_accumulator with raw m³", func(t *testing.T) {
		inv := meterInventory(&meterpb.MeterReadingSupport{UsageUnit: "m³"}, dbo.NamingDBO)
		require.Contains(t, inv, dbo.FieldWaterVolumeAccumulator)
		assert.NotContains(t, inv, dbo.FieldEnergyAccumulator)
		assert.Equal(t, "m³", inv[dbo.FieldWaterVolumeAccumulator].Units)
	})

	t.Run("raw naming lists raw point names with native units", func(t *testing.T) {
		inv := meterInventory(&meterpb.MeterReadingSupport{UsageUnit: "kWh", ProducedUnit: "kWh"}, dbo.NamingRaw)
		require.Contains(t, inv, dbo.RawPointUsage)
		require.Contains(t, inv, dbo.RawPointProduced)
		assert.NotContains(t, inv, dbo.FieldEnergyAccumulator)
		assert.Equal(t, "kWh", inv[dbo.RawPointUsage].Units)
	})
}

func TestDiscoveryEvent(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 15, 0, 0, time.UTC)
	inv := map[string]udmi.MetadataPoint{
		dbo.FieldEnergyAccumulator: {Units: "kWh"},
	}

	meta := &metadatapb.Metadata{
		Name: "van/uk/brum/ugs/meters/elec-main",
		Appearance: &metadatapb.Metadata_Appearance{
			Title:       "Main Electrical Meter",
			Description: "Building main power meter",
		},
		Location:   &metadatapb.Metadata_Location{Floor: "03", Zone: "electrical-room"},
		Membership: &metadatapb.Metadata_Membership{Subsystem: "hvac"},
	}

	ev := discoveryEvent(meta, inv, now)

	assert.Equal(t, now, ev.Timestamp)
	assert.Equal(t, udmi.PointsetVersion, ev.Version)
	assert.Equal(t, "Main Electrical Meter", ev.System.Name)
	assert.Equal(t, "Building main power meter", ev.System.Description)
	assert.Equal(t, []string{"hvac"}, ev.System.Tags)
	require.NotNil(t, ev.System.Location)
	assert.Equal(t, "03", ev.System.Location.Floor)

	// site is left empty (Connect enrichment supplies it) and must be omitted, not
	// emitted as an invalid "site":"".
	b, err := json.Marshal(ev)
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"site"`, "empty site must be omitted from discovery")
	require.NotNil(t, ev.Pointset)
	require.Contains(t, ev.Pointset.Points, dbo.FieldEnergyAccumulator)
	assert.Equal(t, "kWh", ev.Pointset.Points[dbo.FieldEnergyAccumulator].Units)
}

func TestDiscoveryEventFallsBackToDeviceName(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 15, 0, 0, time.UTC)

	// no appearance, no location, no membership.
	ev := discoveryEvent(&metadatapb.Metadata{Name: "meter-1"}, nil, now)

	assert.Equal(t, "meter-1", ev.System.Name)
	assert.Empty(t, ev.System.Description)
	assert.Empty(t, ev.System.Tags)
	assert.Nil(t, ev.System.Location)
	assert.Nil(t, ev.Pointset, "no inventory ⇒ no pointset block")
}

// Disk instances are the fnv32a hash of the mount point, pinned here (and in
// Connect's ingest-transform spec) so a change to the hashing can't pass unseen.
const (
	instRoot   = "705468254"  // "/"
	instVarLib = "3086875876" // "/var/lib/bos"
	instC      = "1065500364" // `C:\`
)

func TestResourceUseTelemetry(t *testing.T) {
	t.Run("full reading maps to utilisation and byte points", func(t *testing.T) {
		r := &resourceusepb.ResourceUse{
			Cpu:    &resourceusepb.CpuUse{Utilization: new(float32(12.5)), CorePercent: []float32{10, 15}},
			Memory: &resourceusepb.MemoryUse{Usage: new(uint64(4 << 30)), Limit: new(uint64(16 << 30)), Utilization: new(float32(25))},
			Disks: []*resourceusepb.DiskUse{
				{MountPoint: "/", Usage: new(uint64(100)), FreeBytes: new(uint64(300)), Limit: new(uint64(400)), Utilization: new(float32(25))},
				{MountPoint: "/var/lib/bos", Usage: new(uint64(50)), Limit: new(uint64(200)), Utilization: new(float32(25))},
			},
			Network: &resourceusepb.NetworkUse{ConnectionCount: new(uint64(7))},
		}

		assert.Equal(t, udmi.PointsEvent{
			"cpu_percentage_sensor":                {PresentValue: float32(12.5)},
			"memory_percentage_sensor":             {PresentValue: float32(25)},
			"memory_datasize_sensor":               {PresentValue: uint64(4 << 30)},
			"memory_datasize_capacity":             {PresentValue: uint64(16 << 30)},
			"disk_percentage_sensor_" + instRoot:   {PresentValue: float32(25)},
			"disk_datasize_sensor_" + instRoot:     {PresentValue: uint64(100)},
			"disk_datasize_capacity_" + instRoot:   {PresentValue: uint64(400)},
			"disk_percentage_sensor_" + instVarLib: {PresentValue: float32(25)},
			"disk_datasize_sensor_" + instVarLib:   {PresentValue: uint64(50)},
			"disk_datasize_capacity_" + instVarLib: {PresentValue: uint64(200)},
		}, resourceUseTelemetry(r), "core_percent, free_bytes and connection_count are not exported")
	})

	t.Run("unset optionals are omitted", func(t *testing.T) {
		r := &resourceusepb.ResourceUse{
			Cpu:    &resourceusepb.CpuUse{},
			Memory: &resourceusepb.MemoryUse{Usage: new(uint64(1))},
			Disks:  []*resourceusepb.DiskUse{{MountPoint: "/", Utilization: new(float32(0))}},
		}
		assert.Equal(t, udmi.PointsEvent{
			"memory_datasize_sensor":             {PresentValue: uint64(1)},
			"disk_percentage_sensor_" + instRoot: {PresentValue: float32(0)},
		}, resourceUseTelemetry(r), "a set zero is kept, an unset field is not")
	})

	t.Run("nil reading has no points", func(t *testing.T) {
		assert.Empty(t, resourceUseTelemetry(nil))
	})
}

func TestResourceUseInventory(t *testing.T) {
	r := &resourceusepb.ResourceUse{
		Cpu:    &resourceusepb.CpuUse{Utilization: new(float32(1))},
		Memory: &resourceusepb.MemoryUse{Usage: new(uint64(1)), Limit: new(uint64(2)), Utilization: new(float32(50))},
		Disks:  []*resourceusepb.DiskUse{{MountPoint: `C:\`, Usage: new(uint64(1)), Limit: new(uint64(2)), Utilization: new(float32(50))}},
	}

	assert.Equal(t, map[string]udmi.MetadataPoint{
		"cpu_percentage_sensor":           {Units: "percent"},
		"memory_percentage_sensor":        {Units: "percent"},
		"memory_datasize_sensor":          {Units: "bytes"},
		"memory_datasize_capacity":        {Units: "bytes"},
		"disk_percentage_sensor_" + instC: {Units: "percent", Ref: `C:\`},
		"disk_datasize_sensor_" + instC:   {Units: "bytes", Ref: `C:\`},
		"disk_datasize_capacity_" + instC: {Units: "bytes", Ref: `C:\`},
	}, resourceUseInventory(r))

	assert.Empty(t, resourceUseInventory(nil), "no reading yet ⇒ no inventory")
}

func TestDiskInstance(t *testing.T) {
	assert.Equal(t, instRoot, strconv.FormatUint(uint64(diskInstance("/")), 10))
	assert.Equal(t, instVarLib, strconv.FormatUint(uint64(diskInstance("/var/lib/bos")), 10))
	assert.Equal(t, instC, strconv.FormatUint(uint64(diskInstance(`C:\`)), 10))
	assert.NotEqual(t, diskInstance("/var-lib"), diskInstance("/var/lib"),
		"mounts differing only in punctuation stay distinct")

	// The same mount keeps its points whatever order the disks are listed in.
	a := &resourceusepb.DiskUse{MountPoint: "/var-lib", Utilization: new(float32(1))}
	b := &resourceusepb.DiskUse{MountPoint: "/var/lib", Utilization: new(float32(2))}
	forward := resourceUseTelemetry(&resourceusepb.ResourceUse{Disks: []*resourceusepb.DiskUse{a, b}})
	reverse := resourceUseTelemetry(&resourceusepb.ResourceUse{Disks: []*resourceusepb.DiskUse{b, a}})
	assert.Equal(t, forward, reverse, "names must not depend on disk order")
	assert.Len(t, forward, 2)
}

func TestAirTemperatureTelemetry(t *testing.T) {
	t.Run("ambient readings map to zone air fields", func(t *testing.T) {
		r := &airtemperaturepb.AirTemperature{
			Mode:               airtemperaturepb.AirTemperature_HEAT,
			TemperatureGoal:    &airtemperaturepb.AirTemperature_TemperatureSetPoint{TemperatureSetPoint: &typespb.Temperature{ValueCelsius: 21}},
			AmbientTemperature: &typespb.Temperature{ValueCelsius: 22.5},
			AmbientHumidity:    new(float32(41)),
			DewPoint:           &typespb.Temperature{ValueCelsius: 9},
		}
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldZoneAirTemperatureSensor:      {PresentValue: 22.5},
			dbo.FieldZoneAirRelativeHumiditySensor: {PresentValue: float32(41)},
		}, envTelemetry(r, dbo.NamingDBO, forEachAirTemperaturePoint), "set point, mode and dew point are not exported")
	})

	t.Run("unset optionals are omitted", func(t *testing.T) {
		r := &airtemperaturepb.AirTemperature{AmbientTemperature: &typespb.Temperature{ValueCelsius: 0}}
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldZoneAirTemperatureSensor: {PresentValue: 0.0},
		}, envTelemetry(r, dbo.NamingDBO, forEachAirTemperaturePoint), "a set zero is kept, an unset field is not")
		assert.Empty(t, envTelemetry(&airtemperaturepb.AirTemperature{Mode: airtemperaturepb.AirTemperature_COOL}, dbo.NamingDBO, forEachAirTemperaturePoint))
		assert.Empty(t, envTelemetry(nil, dbo.NamingDBO, forEachAirTemperaturePoint))
	})

	t.Run("raw naming emits the trait's field names", func(t *testing.T) {
		r := &airtemperaturepb.AirTemperature{AmbientTemperature: &typespb.Temperature{ValueCelsius: 22.5}, AmbientHumidity: new(float32(41))}
		assert.Equal(t, udmi.PointsEvent{
			"ambient_temperature": {PresentValue: 22.5},
			"ambient_humidity":    {PresentValue: float32(41)},
		}, envTelemetry(r, dbo.NamingRaw, forEachAirTemperaturePoint))
	})
}

func TestAirQualityTelemetry(t *testing.T) {
	t.Run("readings with a DBO field are exported", func(t *testing.T) {
		r := &airqualitysensorpb.AirQuality{
			CarbonDioxideLevel:              new(float32(612)),
			VolatileOrganicCompounds:        new(float32(0.3)),
			AirPressure:                     new(float32(1013)),
			Comfort:                         airqualitysensorpb.AirQuality_COMFORTABLE,
			InfectionRisk:                   new(float32(5)),
			Score:                           new(float32(90)),
			ParticulateMatter_1:             new(float32(1)),
			ParticulateMatter_25:            new(float32(2.5)),
			ParticulateMatter_10:            new(float32(10)),
			AirChangePerHour:                new(float32(4)),
			VolatileOrganicCompoundsDensity: new(float32(650)),
		}
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldZoneAirCO2ConcentrationSensor: {PresentValue: float32(612)},
			dbo.FieldZoneAirVOCConcentrationSensor: {PresentValue: float32(0.3)},
			dbo.FieldZoneAirVOCDensitySensor:       {PresentValue: float32(650)},
			dbo.FieldAirPressureSensor:             {PresentValue: float32(1013)},
			dbo.FieldZoneAirPM1pt0DensitySensor:    {PresentValue: float32(1)},
			dbo.FieldZoneAirPM2pt5DensitySensor:    {PresentValue: float32(2.5)},
			dbo.FieldZoneAirPM10pt0DensitySensor:   {PresentValue: float32(10)},
		}, envTelemetry(r, dbo.NamingDBO, forEachAirQualityPoint), "comfort, infection risk, score and air changes are not exported")
	})

	t.Run("a CO2-only sensor sends only CO2", func(t *testing.T) {
		r := &airqualitysensorpb.AirQuality{CarbonDioxideLevel: new(float32(0))}
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldZoneAirCO2ConcentrationSensor: {PresentValue: float32(0)},
		}, envTelemetry(r, dbo.NamingDBO, forEachAirQualityPoint))
		assert.Empty(t, envTelemetry(nil, dbo.NamingDBO, forEachAirQualityPoint))
	})

	t.Run("a ppm VOC sensor sends no VOC density", func(t *testing.T) {
		r := &airqualitysensorpb.AirQuality{VolatileOrganicCompounds: new(float32(0.3))}
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldZoneAirVOCConcentrationSensor: {PresentValue: float32(0.3)},
		}, envTelemetry(r, dbo.NamingDBO, forEachAirQualityPoint))
		assert.Equal(t, map[string]udmi.MetadataPoint{
			dbo.FieldZoneAirVOCConcentrationSensor: {Units: "parts_per_million"},
		}, envInventory(r, dbo.NamingDBO, forEachAirQualityPoint))
	})

	t.Run("a VOC-by-mass sensor sends no ppm VOC", func(t *testing.T) {
		r := &airqualitysensorpb.AirQuality{VolatileOrganicCompoundsDensity: new(float32(2650))}
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldZoneAirVOCDensitySensor: {PresentValue: float32(2650)},
		}, envTelemetry(r, dbo.NamingDBO, forEachAirQualityPoint))
		assert.Equal(t, map[string]udmi.MetadataPoint{
			dbo.FieldZoneAirVOCDensitySensor: {Units: "micrograms_per_cubic_meter"},
		}, envInventory(r, dbo.NamingDBO, forEachAirQualityPoint))
	})

	t.Run("raw naming emits the trait's field names", func(t *testing.T) {
		r := &airqualitysensorpb.AirQuality{CarbonDioxideLevel: new(float32(612)), ParticulateMatter_25: new(float32(2.5))}
		assert.Equal(t, udmi.PointsEvent{
			"carbon_dioxide_level":  {PresentValue: float32(612)},
			"particulate_matter_25": {PresentValue: float32(2.5)},
		}, envTelemetry(r, dbo.NamingRaw, forEachAirQualityPoint))
	})
}

func TestOccupancyTelemetry(t *testing.T) {
	const (
		status = dbo.FieldZoneOccupancyStatus
		count  = dbo.FieldZoneOccupancyCount
	)
	var (
		occupied   = udmi.PointValue{PresentValue: dbo.OccupancyOccupied}
		unoccupied = udmi.PointValue{PresentValue: dbo.OccupancyUnoccupied}
		inactive   = udmi.PointValue{PresentValue: dbo.OccupancyInactive}
		unknown    = udmi.PointValue{PresentValue: dbo.OccupancyUnknown}
		people     = func(n int32) udmi.PointValue { return udmi.PointValue{PresentValue: n} }
	)
	for _, tc := range []struct {
		name     string
		counting peopleCounting
		state    occupancysensorpb.Occupancy_State
		count    int32
		want     udmi.PointsEvent
	}{
		// A sensor that says it counts sends its count, as Connect's import reads one.
		{"counter: occupied with a count", countingCounts, occupancysensorpb.Occupancy_OCCUPIED, 4,
			udmi.PointsEvent{status: occupied, count: people(4)}},
		{"counter: a 0 beside occupied is no reading", countingCounts, occupancysensorpb.Occupancy_OCCUPIED, 0,
			udmi.PointsEvent{status: occupied}},
		{"counter: idle is DBO's inactive, and its 0 no reading", countingCounts, occupancysensorpb.Occupancy_IDLE, 0,
			udmi.PointsEvent{status: inactive}},
		{"counter: unoccupied keeps its true zero", countingCounts, occupancysensorpb.Occupancy_UNOCCUPIED, 0,
			udmi.PointsEvent{status: unoccupied, count: people(0)}},
		{"counter: unspecified is DBO's unknown", countingCounts, occupancysensorpb.Occupancy_STATE_UNSPECIFIED, 0,
			udmi.PointsEvent{status: unknown, count: people(0)}},
		{"counter: a state the table doesn't know sends no state", countingCounts, occupancysensorpb.Occupancy_State(99), 2,
			udmi.PointsEvent{count: people(2)}},

		// A sensor that says it doesn't count sends no count, whatever it reports.
		{"non-counter: unoccupied", countingNone, occupancysensorpb.Occupancy_UNOCCUPIED, 0,
			udmi.PointsEvent{status: unoccupied}},
		{"non-counter: a count it reports anyway", countingNone, occupancysensorpb.Occupancy_OCCUPIED, 3,
			udmi.PointsEvent{status: occupied}},

		// An undescribed sensor sends only a count above 0, so a PIR claims no empty room.
		{"undescribed: an empty room's 0 isn't claimed", countingUndescribed, occupancysensorpb.Occupancy_UNOCCUPIED, 0,
			udmi.PointsEvent{status: unoccupied}},
		{"undescribed: occupied with no count", countingUndescribed, occupancysensorpb.Occupancy_OCCUPIED, 0,
			udmi.PointsEvent{status: occupied}},
		{"undescribed: a count above 0 is sent", countingUndescribed, occupancysensorpb.Occupancy_OCCUPIED, 5,
			udmi.PointsEvent{status: occupied, count: people(5)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &occupancysensorpb.Occupancy{State: tc.state, PeopleCount: tc.count, StateChangeTime: timestamppb.Now(), Confidence: 0.9}
			assert.Equal(t, tc.want, envTelemetry(r, dbo.NamingDBO, forEachOccupancyPointFor(tc.counting)))
		})
	}

	t.Run("raw naming sends the trait's state name", func(t *testing.T) {
		r := &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_IDLE, PeopleCount: 3}
		assert.Equal(t, udmi.PointsEvent{
			"state":        {PresentValue: "IDLE"},
			"people_count": {PresentValue: int32(3)},
		}, envTelemetry(r, dbo.NamingRaw, forEachOccupancyPointFor(countingCounts)))
	})

	t.Run("nil reading has no points", func(t *testing.T) {
		assert.Empty(t, envTelemetry(nil, dbo.NamingDBO, forEachOccupancyPointFor(countingCounts)))
	})
}

func TestPeopleCountingOf(t *testing.T) {
	assert.Equal(t, countingUndescribed, peopleCountingOf(nil))
	assert.Equal(t, countingNone, peopleCountingOf(&occupancysensorpb.OccupancySupport{}))
	assert.Equal(t, countingCounts, peopleCountingOf(&occupancysensorpb.OccupancySupport{MaxPeople: 40}))
}

func TestTemperatureTelemetry(t *testing.T) {
	r := &temperaturepb.Temperature{
		SetPoint: &typespb.Temperature{ValueCelsius: 6},
		Measured: &typespb.Temperature{ValueCelsius: 7.5},
	}

	t.Run("measured maps to the generic temperature field", func(t *testing.T) {
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldTemperatureSensor: {PresentValue: 7.5},
		}, envTelemetry(r, dbo.NamingDBO, forEachTemperaturePoint), "the set point is not exported")
		assert.Equal(t, map[string]udmi.MetadataPoint{
			dbo.FieldTemperatureSensor: {Units: dbo.UnitDegreesCelsius},
		}, envInventory(r, dbo.NamingDBO, forEachTemperaturePoint))
	})

	t.Run("raw naming emits the trait's field name", func(t *testing.T) {
		assert.Equal(t, udmi.PointsEvent{"measured": {PresentValue: 7.5}}, envTelemetry(r, dbo.NamingRaw, forEachTemperaturePoint))
		assert.Equal(t, map[string]udmi.MetadataPoint{"measured": {Units: dbo.UnitDegreesCelsius}},
			envInventory(r, dbo.NamingRaw, forEachTemperaturePoint))
	})

	t.Run("a grouped child is keyed by its point in either naming", func(t *testing.T) {
		forEach := forEachTemperaturePointFor("chw_primary_flow")
		for _, n := range []dbo.Naming{dbo.NamingDBO, dbo.NamingRaw} {
			assert.Equal(t, udmi.PointsEvent{"chw_primary_flow": {PresentValue: 7.5}}, envTelemetry(r, n, forEach), "naming %v", n)
			assert.Equal(t, map[string]udmi.MetadataPoint{"chw_primary_flow": {Units: dbo.UnitDegreesCelsius}},
				envInventory(r, n, forEach), "naming %v", n)
		}
	})

	t.Run("no measured reading sends nothing", func(t *testing.T) {
		assert.Empty(t, envTelemetry(&temperaturepb.Temperature{SetPoint: &typespb.Temperature{ValueCelsius: 6}}, dbo.NamingDBO, forEachTemperaturePoint))
		assert.Empty(t, envInventory[*temperaturepb.Temperature](nil, dbo.NamingDBO, forEachTemperaturePoint))
		assert.Equal(t, udmi.PointsEvent{dbo.FieldTemperatureSensor: {PresentValue: 0.0}},
			envTelemetry(&temperaturepb.Temperature{Measured: &typespb.Temperature{}}, dbo.NamingDBO, forEachTemperaturePoint),
			"a measured 0 °C is a reading")
	})
}

func TestLightTelemetry(t *testing.T) {
	conf := func(c float32) *float32 { return &c }
	for _, n := range []dbo.Naming{dbo.NamingDBO, dbo.NamingRaw} {
		name := pointBrightness.name(n)
		want := func(level float32) udmi.PointsEvent { return udmi.PointsEvent{name: {PresentValue: level}} }
		for _, tc := range []struct {
			desc string
			r    *lightpb.Brightness
			want udmi.PointsEvent
		}{
			{"a level", &lightpb.Brightness{LevelPercent: 42}, want(42)},
			{"off is a reading", &lightpb.Brightness{}, want(0)},
			{"full confidence", &lightpb.Brightness{LevelPercent: 42, Confidence: conf(100)}, want(42)},
			{"low confidence is still sent", &lightpb.Brightness{LevelPercent: 42, Confidence: conf(10)}, want(42)},
			{"an unknown level sends nothing", &lightpb.Brightness{LevelPercent: 42, Confidence: conf(0)}, udmi.PointsEvent{}},
			{"no reading sends nothing", nil, udmi.PointsEvent{}},
			{"a preset sends only the level",
				&lightpb.Brightness{LevelPercent: 60, Preset: &lightpb.LightPreset{Name: "bright"}, TargetLevelPercent: 80}, want(60)},
			{"a zero level with an empty preset is off, not ignored",
				&lightpb.Brightness{Preset: &lightpb.LightPreset{}}, want(0)},
		} {
			assert.Equal(t, tc.want, envTelemetry(tc.r, n, forEachLightPoint), "%s (naming %v)", tc.desc, n)
		}
		assert.Equal(t, map[string]udmi.MetadataPoint{name: {Units: dbo.UnitPercent}},
			envInventory(&lightpb.Brightness{LevelPercent: 42}, n, forEachLightPoint), "naming %v", n)
	}
	assert.Equal(t, dbo.FieldBrightnessPercentageSensor, pointBrightness.name(dbo.NamingDBO))
	assert.Equal(t, "level_percent", pointBrightness.name(dbo.NamingRaw))
}

// TestTransportTelemetry is SCB-1542: a lift's or escalator's values go on the fields agreed with
// Connect in SCC-1209, with Connect's multistate vocabulary, in either naming.
func TestTransportTelemetry(t *testing.T) {
	f := func(v float32) *float32 { return &v }
	percent := transportUnitsOf(&transportpb.TransportSupport{LoadUnit: "% of max load"})
	kg := transportUnitsOf(&transportpb.TransportSupport{LoadUnit: "kg", MaxLoad: 1600})
	mmps := transportUnitsOf(&transportpb.TransportSupport{SpeedUnit: "mm/s"})
	door := func(title string, s transportpb.Transport_Door_DoorStatus) *transportpb.Transport_Door {
		return &transportpb.Transport_Door{Title: title, Status: s}
	}
	at := func(floor string) *transportpb.Transport_Location {
		return &transportpb.Transport_Location{Floor: floor}
	}

	for _, tc := range []struct {
		desc     string
		r        *transportpb.Transport
		u        transportUnits
		dbo, raw udmi.PointsEvent
	}{
		{
			desc: "a 40LH lift",
			r: &transportpb.Transport{
				ActualPosition:   at("12"),
				MovingDirection:  transportpb.Transport_UP,
				Doors:            []*transportpb.Transport_Door{door("Side 1", transportpb.Transport_Door_OPENING)},
				Load:             f(35),
				NextDestinations: []*transportpb.Transport_Location{at("14")},
			},
			u: percent,
			dbo: udmi.PointsEvent{
				dbo.FieldFloorCount:           {PresentValue: 12},
				dbo.FieldDirectionStatus:      {PresentValue: "up"},
				dbo.FieldPositionStatus:       {PresentValue: "opening"},
				dbo.FieldLoadPercentageSensor: {PresentValue: float32(35)},
			},
			raw: udmi.PointsEvent{
				"actual_position":  {PresentValue: 12},
				"moving_direction": {PresentValue: "up"},
				"doors":            {PresentValue: "opening"},
				"load":             {PresentValue: float32(35)},
			},
		},
		{
			desc: "a 40LH escalator",
			r:    &transportpb.Transport{MovingDirection: transportpb.Transport_DOWN, Speed: f(500)},
			u:    mmps,
			dbo: udmi.PointsEvent{
				dbo.FieldDirectionStatus:      {PresentValue: "down"},
				dbo.FieldLinearVelocitySensor: {PresentValue: float32(500)},
			},
			raw: udmi.PointsEvent{
				"moving_direction": {PresentValue: "down"},
				"speed":            {PresentValue: float32(500)},
			},
		},
		{
			desc: "not moving is stopped, and a stopped escalator's speed is a reading",
			r:    &transportpb.Transport{MovingDirection: transportpb.Transport_NO_DIRECTION, Speed: f(0)},
			u:    mmps,
			dbo: udmi.PointsEvent{
				dbo.FieldDirectionStatus:      {PresentValue: "stopped"},
				dbo.FieldLinearVelocitySensor: {PresentValue: float32(0)},
			},
			raw: udmi.PointsEvent{"moving_direction": {PresentValue: "stopped"}, "speed": {PresentValue: float32(0)}},
		},
		{
			desc: "another direction goes as its own name",
			r:    &transportpb.Transport{MovingDirection: transportpb.Transport_FORWARD},
			dbo:  udmi.PointsEvent{dbo.FieldDirectionStatus: {PresentValue: "forward"}},
			raw:  udmi.PointsEvent{"moving_direction": {PresentValue: "forward"}},
		},
		{
			desc: "unspecified states send nothing",
			r: &transportpb.Transport{
				Doors: []*transportpb.Transport_Door{door("Side 1", transportpb.Transport_Door_DOOR_STATUS_UNSPECIFIED)},
			},
			dbo: udmi.PointsEvent{},
			raw: udmi.PointsEvent{},
		},
		{
			desc: "an enum number with no name sends nothing",
			r:    &transportpb.Transport{MovingDirection: 99, OperatingMode: 99},
			dbo:  udmi.PointsEvent{},
			raw:  udmi.PointsEvent{},
		},
		{
			desc: "a float node's basement floor",
			r:    &transportpb.Transport{ActualPosition: at("-3.00")},
			dbo:  udmi.PointsEvent{dbo.FieldFloorCount: {PresentValue: -3}},
			raw:  udmi.PointsEvent{"actual_position": {PresentValue: -3}},
		},
		{
			desc: "ground is floor 0",
			r:    &transportpb.Transport{ActualPosition: at("0")},
			dbo:  udmi.PointsEvent{dbo.FieldFloorCount: {PresentValue: 0}},
			raw:  udmi.PointsEvent{"actual_position": {PresentValue: 0}},
		},
		{
			desc: "a named floor isn't sent",
			r:    &transportpb.Transport{ActualPosition: at("GF")},
			dbo:  udmi.PointsEvent{},
			raw:  udmi.PointsEvent{},
		},
		{
			desc: "a fractional floor isn't sent",
			r:    &transportpb.Transport{ActualPosition: at("2.50")},
			dbo:  udmi.PointsEvent{},
			raw:  udmi.PointsEvent{},
		},
		{
			desc: "a load in kilograms goes on mass_sensor",
			r:    &transportpb.Transport{Load: f(800)},
			u:    kg,
			dbo:  udmi.PointsEvent{dbo.FieldMassSensor: {PresentValue: float32(800)}},
			raw:  udmi.PointsEvent{"load": {PresentValue: float32(800)}},
		},
		{
			desc: "a load or speed with no unit isn't sent",
			r:    &transportpb.Transport{Load: f(35), Speed: f(500)},
			dbo:  udmi.PointsEvent{},
			raw:  udmi.PointsEvent{},
		},
		{
			desc: "doors are numbered in door order, whatever their titles",
			r: &transportpb.Transport{Doors: []*transportpb.Transport_Door{
				door("Front", transportpb.Transport_Door_CLOSED),
				door("", transportpb.Transport_Door_DOOR_STATUS_UNSPECIFIED),
				door("Front", transportpb.Transport_Door_LIMITED_OPENED),
			}},
			dbo: udmi.PointsEvent{
				dbo.FieldPositionStatus + "_1": {PresentValue: "closed"},
				dbo.FieldPositionStatus + "_3": {PresentValue: "limited_open"},
			},
			raw: udmi.PointsEvent{
				"doors_1": {PresentValue: "closed"},
				"doors_3": {PresentValue: "limited_open"},
			},
		},
		{
			desc: "the operating mode's full name",
			r:    &transportpb.Transport{OperatingMode: transportpb.Transport_FIREFIGHTER_CONTROL},
			dbo:  udmi.PointsEvent{dbo.FieldOperatingMode: {PresentValue: "firefighter_control"}},
			raw:  udmi.PointsEvent{"operating_mode": {PresentValue: "firefighter_control"}},
		},
		{desc: "no reading", dbo: udmi.PointsEvent{}, raw: udmi.PointsEvent{}},
	} {
		forEach := forEachTransportPointFor(tc.u)
		assert.Equal(t, tc.dbo, envTelemetry(tc.r, dbo.NamingDBO, forEach), "%s (dbo naming)", tc.desc)
		assert.Equal(t, tc.raw, envTelemetry(tc.r, dbo.NamingRaw, forEach), "%s (raw naming)", tc.desc)
	}

	assert.Equal(t, map[string]udmi.MetadataPoint{
		dbo.FieldLinearVelocitySensor: {Units: dbo.UnitMillimetersPerSecond},
		dbo.FieldDirectionStatus:      {},
	}, envInventory(&transportpb.Transport{MovingDirection: transportpb.Transport_UP, Speed: f(500)}, dbo.NamingDBO, forEachTransportPointFor(mmps)))
	assert.Equal(t, map[string]udmi.MetadataPoint{
		dbo.FieldLoadPercentageSensor: {Units: dbo.UnitPercent},
	}, envInventory(&transportpb.Transport{Load: f(35)}, dbo.NamingDBO, forEachTransportPointFor(percent)))
	assert.Equal(t, map[string]udmi.MetadataPoint{
		"load": {Units: dbo.UnitKilograms},
	}, envInventory(&transportpb.Transport{Load: f(800)}, dbo.NamingRaw, forEachTransportPointFor(kg)))
}

func TestTransportUnitsOf(t *testing.T) {
	with := func(p envPoint, units string) *envPoint {
		p.units = units
		return &p
	}
	for _, tc := range []struct {
		desc        string
		support     *transportpb.TransportSupport
		load, speed *envPoint
	}{
		{"undescribed", nil, nil, nil},
		{"KONE's percentage", &transportpb.TransportSupport{LoadUnit: "% of max load"}, &pointLoadPercent, nil},
		{"a bare percent sign", &transportpb.TransportSupport{LoadUnit: "%"}, &pointLoadPercent, nil},
		{"percent spelt out", &transportpb.TransportSupport{LoadUnit: " Percent "}, &pointLoadPercent, nil},
		{"kilograms", &transportpb.TransportSupport{LoadUnit: "KG"}, with(pointLoadMass, dbo.UnitKilograms), nil},
		{"pounds", &transportpb.TransportSupport{LoadUnit: "lb"}, with(pointLoadMass, dbo.UnitPoundsMass), nil},
		{"tonnes", &transportpb.TransportSupport{LoadUnit: "t"}, with(pointLoadMass, dbo.UnitMetricTons), nil},
		{"a load unit Connect can't resolve", &transportpb.TransportSupport{LoadUnit: "passengers"}, nil, nil},
		{"a load in a speed unit", &transportpb.TransportSupport{LoadUnit: "m/s"}, nil, nil},
		{"40LH's escalators", &transportpb.TransportSupport{SpeedUnit: "mm/s"}, nil, with(pointSpeed, dbo.UnitMillimetersPerSecond)},
		{"metres per second", &transportpb.TransportSupport{SpeedUnit: "m/s"}, nil, with(pointSpeed, dbo.UnitMetersPerSecond)},
		{"feet per minute", &transportpb.TransportSupport{SpeedUnit: "ft/min"}, nil, with(pointSpeed, dbo.UnitFeetPerMinute)},
		{"a speed unit Connect can't resolve", &transportpb.TransportSupport{SpeedUnit: "steps/min"}, nil, nil},
		{"a speed in a mass unit", &transportpb.TransportSupport{SpeedUnit: "kg"}, nil, nil},
	} {
		u := transportUnitsOf(tc.support)
		assert.Equal(t, tc.load, u.load, tc.desc)
		assert.Equal(t, tc.speed, u.speed, tc.desc)
	}
	assert.Empty(t, transportUnitsOf(nil).declared())
	assert.Equal(t, []envPoint{pointLoadPercent, *with(pointSpeed, dbo.UnitMillimetersPerSecond)},
		transportUnitsOf(&transportpb.TransportSupport{LoadUnit: "%", SpeedUnit: "mm/s"}).declared())
}

func TestPointKey(t *testing.T) {
	for leaf, want := range map[string]string{
		"chw-primary-flow":               "chw_primary_flow",
		"lthw-secondary-return":          "lthw_secondary_return",
		"PHX-X-01-01-primary-lthw-flow":  "phx_x_01_01_primary_lthw_flow",
		"DHW-N-09-PHX-primary-lthw-flow": "dhw_n_09_phx_primary_lthw_flow",
		"  Flow  Temp (°C) ":             "flow_temp_c",
		"--a__b--":                       "a_b",
		"already_snake_1":                "already_snake_1",
		"°/·":                            "",
	} {
		assert.Equal(t, want, pointKey(leaf), "pointKey(%q)", leaf)
	}
}

func TestGroupTarget(t *testing.T) {
	known := func(names ...string) func(string) bool {
		return func(n string) bool { return slices.Contains(names, n) }
	}
	const parent = "uk-40lh/floors/01/devices/BMS-S-01-02-PHX-S-01-01"

	target, leaf, ok := groupTarget(parent+"/chw-primary-flow", known(parent))
	assert.True(t, ok)
	assert.Equal(t, parent, target)
	assert.Equal(t, "chw-primary-flow", leaf)

	_, _, ok = groupTarget(parent+"/chw-primary-flow", known())
	assert.False(t, ok, "a parent that isn't announced")
	_, _, ok = groupTarget(parent, known("uk-40lh/floors/01/devices"))
	assert.True(t, ok, "any announced parent, whatever its depth")
	_, _, ok = groupTarget("standalone", known(""))
	assert.False(t, ok, "no '/'")
	_, _, ok = groupTarget(parent+"/", known(parent))
	assert.False(t, ok, "an empty leaf")
	_, _, ok = groupTarget("/leaf", known(""))
	assert.False(t, ok, "an empty parent")
}

// TestEnvInventory checks discovery declares every environmental point with the
// unit the trait defines it in, which Connect converts by: it holds a numeric
// reading whose quantity has several units until discovery names one.
func TestEnvInventory(t *testing.T) {
	assert.Equal(t, map[string]udmi.MetadataPoint{
		dbo.FieldZoneAirTemperatureSensor:      {Units: "degrees_celsius"},
		dbo.FieldZoneAirRelativeHumiditySensor: {Units: "percent_relative_humidity"},
	}, envInventory(&airtemperaturepb.AirTemperature{
		AmbientTemperature: &typespb.Temperature{ValueCelsius: 22.5},
		AmbientHumidity:    new(float32(41)),
	}, dbo.NamingDBO, forEachAirTemperaturePoint))

	assert.Equal(t, map[string]udmi.MetadataPoint{
		dbo.FieldZoneAirCO2ConcentrationSensor: {Units: "parts_per_million"},
		dbo.FieldZoneAirVOCConcentrationSensor: {Units: "parts_per_million"},
		dbo.FieldZoneAirVOCDensitySensor:       {Units: "micrograms_per_cubic_meter"},
		dbo.FieldAirPressureSensor:             {Units: "hectopascals"},
		dbo.FieldZoneAirPM1pt0DensitySensor:    {Units: "micrograms_per_cubic_meter"},
		dbo.FieldZoneAirPM2pt5DensitySensor:    {Units: "micrograms_per_cubic_meter"},
		dbo.FieldZoneAirPM10pt0DensitySensor:   {Units: "micrograms_per_cubic_meter"},
	}, envInventory(&airqualitysensorpb.AirQuality{
		CarbonDioxideLevel:              new(float32(1)),
		VolatileOrganicCompounds:        new(float32(1)),
		AirPressure:                     new(float32(1)),
		ParticulateMatter_1:             new(float32(1)),
		ParticulateMatter_25:            new(float32(1)),
		ParticulateMatter_10:            new(float32(1)),
		VolatileOrganicCompoundsDensity: new(float32(1)),
	}, dbo.NamingDBO, forEachAirQualityPoint))

	assert.Equal(t, map[string]udmi.MetadataPoint{
		dbo.FieldZoneOccupancyStatus: {},
		dbo.FieldZoneOccupancyCount:  {},
	}, envInventory(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED}, dbo.NamingDBO, forEachOccupancyPointFor(countingCounts)),
		"a multistate and a headcount have no unit")

	assert.Equal(t, map[string]udmi.MetadataPoint{
		"ambient_temperature": {Units: "degrees_celsius"},
	}, envInventory(&airtemperaturepb.AirTemperature{AmbientTemperature: &typespb.Temperature{}}, dbo.NamingRaw, forEachAirTemperaturePoint),
		"raw naming keeps the units")

	assert.Empty(t, envInventory[*airqualitysensorpb.AirQuality](nil, dbo.NamingDBO, forEachAirQualityPoint), "no reading yet ⇒ no inventory")
}
