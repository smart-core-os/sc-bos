package protov1

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

// TestProtomove runs integration tests using txtar fixtures
func TestProtomove(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"basic scenarios", "testdata/basic_scenarios.txtar"},
		{"proto with imports", "testdata/with_imports.txtar"},
		{"related files", "testdata/related_files.txtar"},
		{"standalone with underscore", "testdata/standalone_with_underscore.txtar"},
		{"external trait import", "testdata/external_trait_import.txtar"},
		{"type qualification", "testdata/type_qualification.txtar"},
		{"nested packages", "testdata/non_standard_package.txtar"},
		{"non-root protos", "testdata/non_root_protos.txtar"},
		{"no changes needed", "testdata/no_changes_needed.txtar"},
		{"import aware types", "testdata/import_aware_types.txtar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixtest.Run(t, tt.file, run)
		})
	}
}
