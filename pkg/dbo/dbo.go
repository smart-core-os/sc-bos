// Package dbo maps Smart Core trait data onto Google's Digital Buildings Ontology
// (DBO): standard field names, unit names, and entity types.
//
// UDMI is the telemetry wire format DBO consumes; DBO supplies the semantic field
// names, which a building-config "translation" binds to a device's raw points. By
// emitting DBO standard field names directly (we control the names on the
// trait-poll path), that translation collapses to an identity mapping.
//
// See .claude/plans/dbo-conformance-plan.md and the DBO ontology
// (google/digitalbuildings): fields in ontology/yaml/resources/fields, units in
// ontology/yaml/resources/units, meter types under resources/METERS.
package dbo

import "strings"

// DBO standard field names (ontology/yaml/resources/fields/telemetry_fields.yaml).
const (
	// FieldEnergyAccumulator is cumulative (imported/consumed) energy — point type
	// "accumulator" = a total accumulated quantity.
	FieldEnergyAccumulator = "energy_accumulator"
	// FieldExportedEnergyAccumulator is cumulative exported/produced energy.
	FieldExportedEnergyAccumulator = "exported_energy_accumulator"
	// FieldPowerSensor is instantaneous real power. Not produced by the meter trait
	// (which reports cumulative energy only); named here for completeness.
	FieldPowerSensor = "power_sensor"
)

// DBO unit names (ontology/yaml/resources/units/units.yaml). Energy's SI standard
// is joules and power's is watts; the kWh/kW forms below are accepted alternatives.
const (
	UnitKilowattHours = "kilowatt_hours"
	UnitWattHours     = "watt_hours"
	UnitMegawattHours = "megawatt_hours"
	UnitWatts         = "watts"
	UnitKilowatts     = "kilowatts"
)

// PlaceholderMeterType is emitted as the DBO entity type for an energy-only meter.
// It is deliberately NOT a canonical DBO type: every canonical METERS/EM_PWM* type
// requires a power_sensor field (inherited from the global /PWM abstract), which a
// meter reporting only cumulative energy does not have. Resolve this before the
// building-config is treated as authoritative (propose an energy-only DBO type, use
// a passthrough type with allow_undefined_fields, or add power reporting).
const PlaceholderMeterType = "TODO_ENERGY_METER_NO_CANONICAL_TYPE"

// UnitName maps a raw device unit string to its DBO unit name, or "" if there is no
// known mapping (the caller should then fall back to the raw string). Matching is
// case-insensitive over the common textual forms.
func UnitName(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "kwh", "kilowatt_hours", "kilowatt-hours", "kilowatthours":
		return UnitKilowattHours
	case "wh", "watt_hours", "watt-hours", "watthours":
		return UnitWattHours
	case "mwh", "megawatt_hours", "megawatt-hours", "megawatthours":
		return UnitMegawattHours
	case "w", "watts", "watt":
		return UnitWatts
	case "kw", "kilowatts", "kilowatt":
		return UnitKilowatts
	default:
		return ""
	}
}

// MeterField is one DBO point a meter exposes: the standard field name, the raw
// unit string the device reports, its DBO unit name (empty if unmappable), and
// whether it is the exported/produced reading (vs consumed).
type MeterField struct {
	Field    string
	RawUnit  string
	Unit     string
	Exported bool
}

// MeterFields returns the DBO fields an electricity meter exposes given the units
// declared by its reading support. energy_accumulator is always present; the
// exported_energy_accumulator is present only when the meter declares a produced
// unit (i.e. it actually reports export), avoiding a spurious constant-zero series.
func MeterFields(usageUnit, producedUnit string) []MeterField {
	fields := []MeterField{{
		Field:   FieldEnergyAccumulator,
		RawUnit: usageUnit,
		Unit:    UnitName(usageUnit),
	}}
	if producedUnit != "" {
		fields = append(fields, MeterField{
			Field:    FieldExportedEnergyAccumulator,
			RawUnit:  producedUnit,
			Unit:     UnitName(producedUnit),
			Exported: true,
		})
	}
	return fields
}

// MeterEntityType returns the DBO entity type to assign a meter, and whether it is
// a canonical DBO type. For an energy-only meter it is always the non-canonical
// placeholder (see PlaceholderMeterType).
func MeterEntityType() (name string, canonical bool) {
	return PlaceholderMeterType, false
}
