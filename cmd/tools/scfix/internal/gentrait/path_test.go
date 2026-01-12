package gentrait

import (
	"testing"
)

func TestGentraitToProtoImportPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Standard cases - add 'pb' suffix
		{
			name:     "simple trait without pb suffix",
			input:    "meter",
			expected: "meterpb",
		},
		{
			name:     "trait already has pb suffix",
			input:    "wastepb",
			expected: "wastepb",
		},
		{
			name:     "trait with subdirectory",
			input:    "healthpb/utils",
			expected: "healthpb/utils",
		},
		{
			name:     "trait with multiple subdirectories",
			input:    "healthpb/standard/common",
			expected: "healthpb/standard/common",
		},

		// Special case mappings
		{
			name:     "dalipb maps to driver/dalipb",
			input:    "dalipb",
			expected: "driver/dalipb",
		},
		{
			name:     "dalipb with subdirectory preserves structure",
			input:    "dalipb/model",
			expected: "driver/dalipb/model",
		},
		{
			name:     "lighttest maps to lightingtestpb",
			input:    "lighttest",
			expected: "lightingtestpb",
		},
		{
			name:     "servicepb maps to servicespb (plural)",
			input:    "servicepb",
			expected: "servicespb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gentraitToProtoImportPath(tt.input)
			if result != tt.expected {
				t.Errorf("gentraitToProtoImportPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRelPathToProtoImport(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedPkg    string
		expectedImport string
	}{
		{
			name:           "standard trait file",
			input:          "meter/info.go",
			expectedPkg:    "meterpb",
			expectedImport: "github.com/smart-core-os/sc-bos/pkg/proto/meterpb",
		},
		{
			name:           "trait with subdirectory",
			input:          "healthpb/utils/util.go",
			expectedPkg:    "utils",
			expectedImport: "github.com/smart-core-os/sc-bos/pkg/proto/healthpb/utils",
		},
		{
			name:           "dalipb special case",
			input:          "dalipb/dalipb.go",
			expectedPkg:    "dalipb",
			expectedImport: "github.com/smart-core-os/sc-bos/pkg/proto/driver/dalipb",
		},
		{
			name:           "lighttest special case",
			input:          "lighttest/holder.go",
			expectedPkg:    "lightingtestpb",
			expectedImport: "github.com/smart-core-os/sc-bos/pkg/proto/lightingtestpb",
		},
		{
			name:           "servicepb special case",
			input:          "servicepb/rename.go",
			expectedPkg:    "servicespb",
			expectedImport: "github.com/smart-core-os/sc-bos/pkg/proto/servicespb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, importPath := relPathToProtoImport(tt.input)
			if pkg != tt.expectedPkg {
				t.Errorf("relPathToProtoImport(%q) package = %q, want %q", tt.input, pkg, tt.expectedPkg)
			}
			if importPath != tt.expectedImport {
				t.Errorf("relPathToProtoImport(%q) import = %q, want %q", tt.input, importPath, tt.expectedImport)
			}
		})
	}
}
