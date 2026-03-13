package scgolang

import "testing"

func TestScgolangImportToScBos(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
		wantOK   bool
	}{
		// Non sc-golang imports are returned unchanged.
		{name: "non_sc_golang", input: "github.com/some/other/pkg", wantPath: "github.com/some/other/pkg", wantOK: false},
		{name: "stdlib", input: "fmt", wantPath: "fmt", wantOK: false},

		// Removed packages — silently skipped.
		{name: "removed_server", input: oldModule + "/pkg/server", wantPath: "", wantOK: false},
		{name: "removed_middleware", input: oldModule + "/pkg/middleware/grpc", wantPath: "", wantOK: false},
		{name: "removed_client", input: oldModule + "/pkg/client", wantPath: "", wantOK: false},

		// internal/minibus -> pkg/minibus
		{name: "minibus", input: oldModule + "/internal/minibus", wantPath: newModule + "/pkg/minibus", wantOK: true},

		// pkg/time -> pkg/util/time
		{name: "time", input: oldModule + "/pkg/time", wantPath: newModule + "/pkg/util/time", wantOK: true},
		{name: "time_sub", input: oldModule + "/pkg/time/sub", wantPath: newModule + "/pkg/util/time/sub", wantOK: true},

		// pkg/masks -> pkg/util/masks
		{name: "masks", input: oldModule + "/pkg/masks", wantPath: newModule + "/pkg/util/masks", wantOK: true},

		// pkg/cmp -> pkg/util/cmp
		{name: "cmp", input: oldModule + "/pkg/cmp", wantPath: newModule + "/pkg/util/cmp", wantOK: true},

		// pkg/trait (exact) -> pkg/trait
		{name: "trait_exact", input: oldModule + "/pkg/trait", wantPath: newModule + "/pkg/trait", wantOK: true},

		// Conflicting trait packages -> pkg/trait/
		{name: "conflicting_accesspb", input: oldModule + "/pkg/trait/accesspb", wantPath: newModule + "/pkg/trait/accesspb", wantOK: true},
		{name: "conflicting_airqualitysensorpb", input: oldModule + "/pkg/trait/airqualitysensorpb", wantPath: newModule + "/pkg/trait/airqualitysensorpb", wantOK: true},
		{name: "conflicting_airtemperaturepb", input: oldModule + "/pkg/trait/airtemperaturepb", wantPath: newModule + "/pkg/trait/airtemperaturepb", wantOK: true},
		{name: "conflicting_electricpb", input: oldModule + "/pkg/trait/electricpb", wantPath: newModule + "/pkg/trait/electricpb", wantOK: true},
		{name: "conflicting_enterleavesensorpb", input: oldModule + "/pkg/trait/enterleavesensorpb", wantPath: newModule + "/pkg/trait/enterleavesensorpb", wantOK: true},
		{name: "conflicting_meterpb", input: oldModule + "/pkg/trait/meterpb", wantPath: newModule + "/pkg/trait/meterpb", wantOK: true},
		{name: "conflicting_occupancysensorpb", input: oldModule + "/pkg/trait/occupancysensorpb", wantPath: newModule + "/pkg/trait/occupancysensorpb", wantOK: true},
		{name: "conflicting_temperaturepb", input: oldModule + "/pkg/trait/temperaturepb", wantPath: newModule + "/pkg/trait/temperaturepb", wantOK: true},
		{name: "conflicting_wastepb", input: oldModule + "/pkg/trait/wastepb", wantPath: newModule + "/pkg/trait/wastepb", wantOK: true},

		// Non-conflicting trait packages -> pkg/proto/
		{name: "lightpb", input: oldModule + "/pkg/trait/lightpb", wantPath: newModule + "/pkg/proto/lightpb", wantOK: true},
		{name: "onoffpb", input: oldModule + "/pkg/trait/onoffpb", wantPath: newModule + "/pkg/proto/onoffpb", wantOK: true},
		{name: "bookingpb", input: oldModule + "/pkg/trait/bookingpb", wantPath: newModule + "/pkg/proto/bookingpb", wantOK: true},
		{name: "brightnesssensorpb", input: oldModule + "/pkg/trait/brightnesssensorpb", wantPath: newModule + "/pkg/proto/brightnesssensorpb", wantOK: true},
		{name: "vendingpb_unitpb", input: oldModule + "/pkg/trait/vendingpb/unitpb", wantPath: newModule + "/pkg/proto/vendingpb/unitpb", wantOK: true},
		{name: "electricpb_modepb", input: oldModule + "/pkg/trait/electricpb/modepb", wantPath: newModule + "/pkg/trait/electricpb/modepb", wantOK: true},

		// Identity-mapped packages — just swap module prefix.
		{name: "resource", input: oldModule + "/pkg/resource", wantPath: newModule + "/pkg/resource", wantOK: true},
		{name: "router", input: oldModule + "/pkg/router", wantPath: newModule + "/pkg/router", wantOK: true},
		{name: "group", input: oldModule + "/pkg/group", wantPath: newModule + "/pkg/group", wantOK: true},
		{name: "wrap", input: oldModule + "/pkg/wrap", wantPath: newModule + "/pkg/wrap", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotOK := scgolangImportToScBos(tt.input)
			if gotOK != tt.wantOK {
				t.Errorf("ok: got %v want %v", gotOK, tt.wantOK)
			}
			if gotPath != tt.wantPath {
				t.Errorf("path: got %q want %q", gotPath, tt.wantPath)
			}
		})
	}
}
