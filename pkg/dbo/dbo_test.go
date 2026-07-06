package dbo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitName(t *testing.T) {
	cases := map[string]string{
		"kWh":            UnitKilowattHours,
		"kilowatt-hours": UnitKilowattHours,
		"KWH":            UnitKilowattHours,
		"Wh":             UnitWattHours,
		"MWh":            UnitMegawattHours,
		"kW":             UnitKilowatts,
		"W":              UnitWatts,
		" kWh ":          UnitKilowattHours,
		"furlongs":       "", // unknown → empty
		"":               "",
	}
	for raw, want := range cases {
		assert.Equalf(t, want, UnitName(raw), "UnitName(%q)", raw)
	}
}

func TestMeterFields(t *testing.T) {
	t.Run("consumption only", func(t *testing.T) {
		fields := MeterFields("kWh", "")
		require.Len(t, fields, 1)
		assert.Equal(t, FieldEnergyAccumulator, fields[0].Field)
		assert.Equal(t, "kWh", fields[0].RawUnit)
		assert.Equal(t, UnitKilowattHours, fields[0].Unit)
		assert.False(t, fields[0].Exported)
	})

	t.Run("consumption and export", func(t *testing.T) {
		fields := MeterFields("kWh", "kWh")
		require.Len(t, fields, 2)
		assert.Equal(t, FieldEnergyAccumulator, fields[0].Field)
		assert.Equal(t, FieldExportedEnergyAccumulator, fields[1].Field)
		assert.True(t, fields[1].Exported)
		assert.Equal(t, UnitKilowattHours, fields[1].Unit)
	})

	t.Run("unmappable unit keeps raw, empty dbo unit", func(t *testing.T) {
		fields := MeterFields("blips", "")
		require.Len(t, fields, 1)
		assert.Equal(t, "blips", fields[0].RawUnit)
		assert.Empty(t, fields[0].Unit)
	})
}

func TestMeterEntityTypeIsPlaceholder(t *testing.T) {
	name, canonical := MeterEntityType()
	assert.Equal(t, PlaceholderMeterType, name)
	assert.False(t, canonical, "energy-only meter has no canonical DBO type")
}
