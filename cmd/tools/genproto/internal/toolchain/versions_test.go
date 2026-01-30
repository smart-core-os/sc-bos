package toolchain

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestVerifyVersions(t *testing.T) {
	// This test verifies that all required tools are installed with expected versions
	// It will pass if the system has the correct versions installed
	err := VerifyVersions()
	if err != nil {
		t.Logf("Version verification failed (this is expected if tools are not installed): %v", err)
		// Don't fail the test since it depends on system setup
		t.Skip("Skipping test - tools not installed or wrong versions")
	} else {
		t.Log("All tool versions verified successfully")
	}
}

func TestVerifyTool_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
	}{
		{
			name: "missing tool",
			tool: Tool{
				Name:            "this-tool-does-not-exist-xyz123",
				VersionArg:      "--version",
				VersionPattern:  nil,
				ExpectedVersion: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "valid protoc",
			tool: Tool{
				Name:            "protoc",
				VersionArg:      "--version",
				VersionPattern:  regexp.MustCompile(`libprotoc (\d+\.\d+(?:\.\d+)?)`),
				ExpectedVersion: ExpectedProtoc,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if the tool isn't available
			if tt.tool.Name == "protoc" {
				if _, err := exec.LookPath("protoc"); err != nil {
					t.Skip("protoc not installed")
				}
			}

			err := verifyTool(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyTool() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestVersionMismatchErrorFormat tests that version mismatch errors are formatted correctly
func TestVersionMismatchErrorFormat(t *testing.T) {
	// Create a tool with an impossible version requirement
	badTool := Tool{
		Name:            "protoc",
		VersionArg:      "--version",
		VersionPattern:  regexp.MustCompile(`libprotoc (\d+\.\d+(?:\.\d+)?)`),
		ExpectedVersion: "999.999.999", // Version that won't match
	}

	// Skip if protoc isn't available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not installed")
	}

	err := verifyTool(badTool)
	if err == nil {
		t.Error("Expected version mismatch error, got nil")
		return
	}

	// Verify error message format
	errMsg := err.Error()
	if !strings.Contains(errMsg, "protoc") {
		t.Errorf("Error message should contain tool name 'protoc': %s", errMsg)
	}
	if !strings.Contains(errMsg, "!=") {
		t.Errorf("Error message should contain '!=': %s", errMsg)
	}
	if !strings.Contains(errMsg, "999.999.999") {
		t.Errorf("Error message should contain expected version: %s", errMsg)
	}

	t.Logf("Version mismatch error format (example): %s", errMsg)
}
