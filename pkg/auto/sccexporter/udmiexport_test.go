package sccexporter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
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

	t.Run("usage and produced with support", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 123.45, Produced: 67.89, EndTime: timestamppb.New(end)}
		support := &meterpb.MeterReadingSupport{UsageUnit: "kWh", ProducedUnit: "kWh"}

		ev := meterTelemetry(r, support, now)

		assert.Equal(t, end, ev.Timestamp, "timestamp should be the reading end_time")
		assert.Equal(t, udmi.PointsetVersion, ev.Version)
		require.Contains(t, ev.Points, meterPointUsage)
		require.Contains(t, ev.Points, meterPointProduced)
		assert.Equal(t, float32(123.45), ev.Points[meterPointUsage].PresentValue)
		assert.Equal(t, float32(67.89), ev.Points[meterPointProduced].PresentValue)
	})

	t.Run("falls back to now when end_time unset", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 1}
		ev := meterTelemetry(r, &meterpb.MeterReadingSupport{UsageUnit: "kWh"}, now)
		assert.Equal(t, now, ev.Timestamp)
	})

	t.Run("consumption-only meter omits produced when support has no producedUnit", func(t *testing.T) {
		r := &meterpb.MeterReading{Usage: 10, Produced: 0, EndTime: timestamppb.New(end)}
		support := &meterpb.MeterReadingSupport{UsageUnit: "kWh"} // no ProducedUnit

		ev := meterTelemetry(r, support, now)

		require.Contains(t, ev.Points, meterPointUsage)
		assert.NotContains(t, ev.Points, meterPointProduced, "no producedUnit ⇒ no produced series")
	})

	t.Run("without support falls back to non-zero produced", func(t *testing.T) {
		withProduced := meterTelemetry(&meterpb.MeterReading{Usage: 10, Produced: 5}, nil, now)
		assert.Contains(t, withProduced.Points, meterPointProduced)

		zeroProduced := meterTelemetry(&meterpb.MeterReading{Usage: 10, Produced: 0}, nil, now)
		assert.NotContains(t, zeroProduced.Points, meterPointProduced)
	})
}

func TestMeterInventory(t *testing.T) {
	t.Run("usage and produced units", func(t *testing.T) {
		inv := meterInventory(&meterpb.MeterReadingSupport{UsageUnit: "kWh", ProducedUnit: "kWh"})
		require.Contains(t, inv, meterPointUsage)
		require.Contains(t, inv, meterPointProduced)
		assert.Equal(t, "kWh", inv[meterPointUsage].Units)
		assert.Equal(t, "kWh", inv[meterPointProduced].Units)
		// meters are read-only: nothing writable.
		assert.False(t, inv[meterPointUsage].Writable)
		assert.False(t, inv[meterPointProduced].Writable)
	})

	t.Run("usage only when no producedUnit", func(t *testing.T) {
		inv := meterInventory(&meterpb.MeterReadingSupport{UsageUnit: "kWh"})
		require.Contains(t, inv, meterPointUsage)
		assert.NotContains(t, inv, meterPointProduced)
	})

	t.Run("nil support still lists usage", func(t *testing.T) {
		inv := meterInventory(nil)
		require.Contains(t, inv, meterPointUsage)
		assert.Empty(t, inv[meterPointUsage].Units)
	})
}

func TestDiscoveryEvent(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 15, 0, 0, time.UTC)
	inv := map[string]udmi.MetadataPoint{
		meterPointUsage: {Units: "kWh"},
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
	require.NotNil(t, ev.Pointset)
	require.Contains(t, ev.Pointset.Points, meterPointUsage)
	assert.Equal(t, "kWh", ev.Pointset.Points[meterPointUsage].Units)
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
