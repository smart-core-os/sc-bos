package toolchain

import (
	"strings"
	"testing"
)

func TestRunProtoc(t *testing.T) {
	// Test that we can run protoc --version
	err := RunProtoc("", "--version")
	if err != nil {
		t.Fatalf("RunProtoc(--version) failed: %v", err)
	}
}

func TestRunProtoc_InvalidCommand(t *testing.T) {
	// Test that invalid invocations return an error
	err := RunProtoc("", "--invalid-flag-that-does-not-exist")
	if err == nil {
		t.Fatal("RunProtoc with invalid flag should return an error")
	}

	// Error should mention protoc
	if !strings.Contains(err.Error(), "protoc") {
		t.Errorf("Expected error to mention 'protoc', got: %v", err)
	}
}
