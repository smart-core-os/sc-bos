package connecttelemetry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
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
