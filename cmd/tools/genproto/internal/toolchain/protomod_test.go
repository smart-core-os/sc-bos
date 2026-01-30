package toolchain

import (
	"strings"
	"testing"
)

func TestRunProtomod(t *testing.T) {
	// Test that we can run protomod --help
	err := RunProtomod("", "--help")
	if err != nil {
		t.Fatalf("RunProtomod(--help) failed: %v", err)
	}
}

func TestRunProtomod_InvalidCommand(t *testing.T) {
	// Test that invalid commands return an error
	err := RunProtomod("", "invalid-command-that-does-not-exist")
	if err == nil {
		t.Fatal("RunProtomod with invalid command should return an error")
	}

	// Error should mention protomod
	if !strings.Contains(err.Error(), "protomod") {
		t.Errorf("Expected error to mention 'protomod', got: %v", err)
	}
}
