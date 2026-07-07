// Package dbo maps Smart Core trait data onto Google's Digital Buildings Ontology
// (DBO): standard field names, unit names, and entity types.
//
// UDMI is the telemetry wire format DBO consumes; DBO supplies the semantic field
// names, which a building-config "translation" binds to a device's raw points. By
// emitting DBO standard field names directly (we control the names on the
// trait-poll path), that translation collapses to an identity mapping.
//
// The Meter trait carries no medium/commodity signal — only the reading's unit — so
// a meter's commodity is inferred from its usage unit: energy units (kWh/Wh/MWh) ⇒
// an electricity meter, cubic metres ⇒ a water meter. See commodityForUnit.
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
	// FieldWaterVolumeAccumulator is cumulative water volume — the single point a
	// standard building water meter (METERS/WM_STANDARD) reports.
	FieldWaterVolumeAccumulator = "water_volume_accumulator"
)

// DBO unit names (ontology/yaml/resources/units/units.yaml). Energy's SI standard
// is joules and power's is watts; the kWh/kW forms below are accepted alternatives.
// cubic_meters is the standard unit of the volume measurement.
const (
	UnitKilowattHours = "kilowatt_hours"
	UnitWattHours     = "watt_hours"
	UnitMegawattHours = "megawatt_hours"
	UnitWatts         = "watts"
	UnitKilowatts     = "kilowatts"
	UnitCubicMeters   = "cubic_meters"
)

// ElectricityMeterType is the DBO entity type for an energy-only electricity meter.
// It is NOT the final canonical classification: every canonical METERS/EM_PWM* type
// requires a power_sensor field (inherited from the global /PWM abstract), which a
// meter reporting only cumulative energy does not have. EM_INITIAL is DBO's "initial"
// electricity-meter type — it implements the EM category with allow_undefined_fields,
// so an energy-only meter validates against it (no power_sensor demanded) while the
// type stays honest that a proper classification (or power reporting) is still to come.
const ElectricityMeterType = "METERS/EM_INITIAL"

// WaterMeterType is the DBO entity type for a standard building water meter. Unlike
// the electricity case it IS canonical: WM_STANDARD's only required field is
// water_volume_accumulator (no power_sensor), which a volume-reporting meter has.
const WaterMeterType = "METERS/WM_STANDARD"

// commodity is the physical quantity a meter measures, inferred from its unit.
type commodity int

const (
	commodityEnergy commodity = iota // electricity: kWh/Wh/MWh (also the default)
	commodityWater                   // water: cubic metres
)

// commodityForUnit infers a meter's commodity from its usage unit.
//
// ASSUMPTION: a cubic-metre unit always means WATER. Volume (m³) is physically
// ambiguous — gas meters also report volume — but Smart Core carries no commodity
// signal on the Meter trait, and this estate has no gas meters (and none are
// anticipated). If gas metering is ever introduced this inference must be revisited
// (e.g. a commodity hint on the trait or in device metadata). Any non-volume unit,
// including unmapped ones, is treated as energy (the electricity default).
func commodityForUnit(usageUnit string) commodity {
	switch normaliseUnit(usageUnit) {
	case "m³", "m3", "cubic_meters", "cubic-meters", "cubicmeters", "cubicmetres", "cubic_metres":
		return commodityWater
	default:
		return commodityEnergy
	}
}

func normaliseUnit(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// UnitName maps a raw device unit string to its DBO unit name, or "" if there is no
// known mapping (the caller should then fall back to the raw string). Matching is
// case-insensitive over the common textual forms.
func UnitName(raw string) string {
	switch normaliseUnit(raw) {
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
	case "m³", "m3", "cubic_meters", "cubic-meters", "cubicmeters", "cubicmetres", "cubic_metres":
		return UnitCubicMeters
	default:
		return ""
	}
}

// Raw Smart Core meter point names — the "vendor" names before DBO normalisation.
// The Meter trait already reduces a meter to these two readings.
const (
	RawPointUsage    = "usage"
	RawPointProduced = "produced"
)

// Naming selects the point key a caller emits on the wire. NamingDBO emits DBO
// standard field names (so a building-config translation is an identity mapping);
// NamingRaw emits the raw Smart Core point names and leaves the name→field mapping
// to the consumer (Connect's per-source pipeline). The zero value is NamingDBO.
type Naming string

const (
	NamingDBO Naming = "dbo"
	NamingRaw Naming = "raw"
)

// MeterField is one DBO point a meter exposes: the DBO standard field name, the raw
// Smart Core point name, the raw unit string the device reports, its DBO unit name
// (empty if unmappable), and whether it is the exported/produced reading (vs consumed).
type MeterField struct {
	Field    string
	RawName  string
	RawUnit  string
	Unit     string
	Exported bool
}

// PointName returns the point key for this field under the given naming mode: the
// raw Smart Core name for NamingRaw, otherwise the DBO standard field name.
func (f MeterField) PointName(n Naming) string {
	if n == NamingRaw {
		return f.RawName
	}
	return f.Field
}

// MeterFields returns the DBO fields a meter exposes given the units declared by its
// reading support. The field set depends on the meter's commodity (inferred from the
// usage unit, see commodityForUnit):
//
//   - electricity: energy_accumulator is always present; exported_energy_accumulator
//     is added only when the meter declares a produced unit (i.e. it actually reports
//     export), avoiding a spurious constant-zero series.
//   - water: a single water_volume_accumulator. Water meters are usage-only — DBO has
//     no exported water field and none of our meters report produced water — so a
//     producedUnit on a water meter is ignored.
func MeterFields(usageUnit, producedUnit string) []MeterField {
	if commodityForUnit(usageUnit) == commodityWater {
		return []MeterField{{
			Field:   FieldWaterVolumeAccumulator,
			RawName: RawPointUsage,
			RawUnit: usageUnit,
			Unit:    UnitName(usageUnit),
		}}
	}

	fields := []MeterField{{
		Field:   FieldEnergyAccumulator,
		RawName: RawPointUsage,
		RawUnit: usageUnit,
		Unit:    UnitName(usageUnit),
	}}
	if producedUnit != "" {
		fields = append(fields, MeterField{
			Field:    FieldExportedEnergyAccumulator,
			RawName:  RawPointProduced,
			RawUnit:  producedUnit,
			Unit:     UnitName(producedUnit),
			Exported: true,
		})
	}
	return fields
}

// MeterEntityType returns the DBO entity type to assign a meter given its usage unit,
// and whether it is the final canonical DBO type. A water meter maps to the canonical
// WM_STANDARD; an energy-only electricity meter has no canonical type and gets the
// EM_INITIAL passthrough (canonical=false, see ElectricityMeterType).
func MeterEntityType(usageUnit string) (name string, canonical bool) {
	if commodityForUnit(usageUnit) == commodityWater {
		return WaterMeterType, true
	}
	return ElectricityMeterType, false
}
