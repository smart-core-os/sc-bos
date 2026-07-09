package dbo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// TestFieldNamesAreValidDBOSubfields is a conformance guard: every DBO standard
// field name this package emits on the wire must decompose, on '_', into subfields
// that exist in Google Digital Buildings' subfield registry. This locks our field
// constants to the real ontology so a typo or an invented name can't ship.
//
// The check mirrors the DBO point-name validation in the JLL validate-udmi tool
// (google/digitalbuildings subfield decomposition). testdata/subfields.yaml is a
// verbatim copy of ontology/yaml/resources/subfields/subfields.yaml; refresh it
// from upstream when bumping the targeted ontology version.
func TestFieldNamesAreValidDBOSubfields(t *testing.T) {
	subfields := loadSubfields(t)

	// Every field name the package can put on the wire. The static constants plus
	// the fields MeterFields actually emits (both usage-only and usage+export), so
	// the guard covers exactly what a meter publishes.
	names := map[string]struct{}{
		FieldEnergyAccumulator:         {},
		FieldExportedEnergyAccumulator: {},
		FieldPowerSensor:               {},
		FieldWaterVolumeAccumulator:    {},
	}
	for _, f := range MeterFields("kWh", "kWh") { // electricity: energy + exported
		names[f.Field] = struct{}{}
	}
	for _, f := range MeterFields("m³", "") { // water: water_volume_accumulator
		names[f.Field] = struct{}{}
	}

	for name := range names {
		if unknown := unknownSegments(name, subfields); len(unknown) > 0 {
			t.Errorf("DBO field %q has segments not in the subfield registry: %v", name, unknown)
		}
	}
}

// loadSubfields reads the vendored DBO subfields.yaml and returns the set of every
// subfield identifier across all top-level sections (aggregation, component,
// descriptor, measurement, point_type, ...).
func loadSubfields(t *testing.T) map[string]struct{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "subfields.yaml"))
	if err != nil {
		t.Fatalf("read subfields.yaml: %v", err)
	}
	var doc map[string]map[string]string
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse subfields.yaml: %v", err)
	}
	out := make(map[string]struct{})
	for _, section := range doc {
		for name := range section {
			out[name] = struct{}{}
		}
	}
	if len(out) == 0 {
		t.Fatal("no subfields parsed from subfields.yaml")
	}
	return out
}

// unknownSegments splits a DBO field name on '_' and returns the segments missing
// from the subfield set (empty result ⇒ every segment is a known subfield).
func unknownSegments(name string, subfields map[string]struct{}) []string {
	var unknown []string
	for _, seg := range strings.Split(name, "_") {
		if _, ok := subfields[seg]; !ok {
			unknown = append(unknown, seg)
		}
	}
	return unknown
}
