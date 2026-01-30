package protofile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	tmpDir := t.TempDir()
	testFiles := []string{
		"test1.proto",
		"test2.proto",
		"subdir/test3.proto",
	}
	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("syntax = \"proto3\";"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}
	// Create a non-proto file that should be ignored
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create non-proto file: %v", err)
	}

	got, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Convert to map for easier comparison
	gotMap := make(map[string]bool)
	for _, f := range got {
		gotMap[f] = true
	}

	if len(got) != len(testFiles) {
		t.Errorf("Discover() found %d files, want %d", len(got), len(testFiles))
	}

	for _, want := range testFiles {
		if !gotMap[want] {
			t.Errorf("Discover() missing expected file: %s", want)
		}
	}

	// Check that non-proto file was not included
	if gotMap["readme.txt"] {
		t.Errorf("Discover() should not include non-proto files")
	}
}

func TestDiscover_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	got, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Discover() in empty directory = %v, want empty slice", got)
	}
}

func TestDiscover_NonExistentDirectory(t *testing.T) {
	_, err := Discover("/nonexistent/directory/path")
	if err == nil {
		t.Error("Discover() with nonexistent directory should return error")
	}
}
