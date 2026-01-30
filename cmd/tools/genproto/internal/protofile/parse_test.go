package protofile

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	skipWithoutProtoc(t)
	// Create a temporary directory with a simple proto file
	tmpDir := t.TempDir()

	protoContent := `syntax = "proto3";

package test;

message TestMessage {
  string name = 1;
  int32 id = 2;
}

service TestService {
  rpc GetTest(TestMessage) returns (TestMessage);
}
`

	protoFile := "test.proto"
	protoPath := filepath.Join(tmpDir, protoFile)

	if err := os.WriteFile(protoPath, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create test proto file: %v", err)
	}

	// Parse the proto file
	fd, err := Parse(tmpDir, protoFile)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify basic properties of the parsed file
	if fd.GetName() != protoFile {
		t.Errorf("Expected file name %q, got %q", protoFile, fd.GetName())
	}

	if fd.GetPackage() != "test" {
		t.Errorf("Expected package %q, got %q", "test", fd.GetPackage())
	}

	// Verify the message exists
	messages := fd.GetMessageType()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	} else {
		if messages[0].GetName() != "TestMessage" {
			t.Errorf("Expected message name %q, got %q", "TestMessage", messages[0].GetName())
		}

		// Verify fields
		fields := messages[0].GetField()
		if len(fields) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(fields))
		}
	}

	// Verify the service exists
	services := fd.GetService()
	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	} else {
		if services[0].GetName() != "TestService" {
			t.Errorf("Expected service name %q, got %q", "TestService", services[0].GetName())
		}
	}
}

func TestParse_NonExistentFile(t *testing.T) {
	skipWithoutProtoc(t)
	tmpDir := t.TempDir()

	// Try to parse a file that doesn't exist
	_, err := Parse(tmpDir, "nonexistent.proto")
	if err == nil {
		t.Fatal("Parse() should fail for nonexistent file")
	}
}

func TestParse_InvalidProto(t *testing.T) {
	skipWithoutProtoc(t)
	tmpDir := t.TempDir()

	// Create a file with invalid proto syntax
	protoFile := "invalid.proto"
	protoPath := filepath.Join(tmpDir, protoFile)

	invalidContent := `this is not valid proto syntax`
	if err := os.WriteFile(protoPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to parse the invalid file
	_, err := Parse(tmpDir, protoFile)
	if err == nil {
		t.Fatal("Parse() should fail for invalid proto file")
	}
}

func TestParse_WithImports(t *testing.T) {
	skipWithoutProtoc(t)
	tmpDir := t.TempDir()

	// Create a base proto file
	baseContent := `syntax = "proto3";

package base;

message BaseMessage {
  string value = 1;
}
`
	baseFile := "base.proto"
	if err := os.WriteFile(filepath.Join(tmpDir, baseFile), []byte(baseContent), 0644); err != nil {
		t.Fatalf("Failed to create base proto file: %v", err)
	}

	// Create a proto file that imports the base
	importingContent := `syntax = "proto3";

package importing;

import "base.proto";

message ImportingMessage {
  base.BaseMessage base = 1;
}
`
	importingFile := "importing.proto"
	if err := os.WriteFile(filepath.Join(tmpDir, importingFile), []byte(importingContent), 0644); err != nil {
		t.Fatalf("Failed to create importing proto file: %v", err)
	}

	// Parse the importing file (should handle imports)
	fd, err := Parse(tmpDir, importingFile)
	if err != nil {
		t.Fatalf("Parse() failed with imports: %v", err)
	}

	if fd.GetName() != importingFile {
		t.Errorf("Expected file name %q, got %q", importingFile, fd.GetName())
	}

	// Verify the dependency is included
	deps := fd.GetDependency()
	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	} else if deps[0] != baseFile {
		t.Errorf("Expected dependency %q, got %q", baseFile, deps[0])
	}
}

// skipWithoutProtoc skips the test if protoc is not installed.
// Parse requires protoc to convert proto files to descriptor format.
func skipWithoutProtoc(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not installed, skipping test")
	}
}
