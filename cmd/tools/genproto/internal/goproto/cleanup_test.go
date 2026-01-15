package goproto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/generator"
)

func TestCleanGeneratedFiles(t *testing.T) {
	archive := loadTxtar(t, "cleanup.txtar")
	tmpDir := t.TempDir()

	// Extract input files
	for _, file := range archive.Files {
		if strings.HasPrefix(file.Name, "input/") {
			relPath := file.Name[len("input/"):]
			path := filepath.Join(tmpDir, relPath)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatalf("creating directory: %v", err)
			}
			if err := os.WriteFile(path, file.Data, 0644); err != nil {
				t.Fatalf("writing file %s: %v", relPath, err)
			}
		}
	}

	ctx := &generator.Context{
		RootDir: tmpDir,
		Config: generator.Config{
			LogLevel: generator.LogLevelQuiet,
			DryRun:   false,
		},
	}

	if err := cleanGeneratedFiles(ctx, tmpDir); err != nil {
		t.Fatalf("cleanGeneratedFiles failed: %v", err)
	}

	// Build expected output files map
	expectedFiles := make(map[string][]byte)
	for _, file := range archive.Files {
		if strings.HasPrefix(file.Name, "output/") {
			relPath := file.Name[len("output/"):]
			expectedFiles[relPath] = file.Data
		}
	}

	// Check all expected files exist with correct content
	for relPath, expectedData := range expectedFiles {
		path := filepath.Join(tmpDir, relPath)
		actualData, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected file %s to exist but got error: %v", relPath, err)
			continue
		}
		if string(actualData) != string(expectedData) {
			t.Errorf("file %s content mismatch:\nexpected:\n%s\nactual:\n%s", relPath, expectedData, actualData)
		}
	}

	// Check that no unexpected files exist
	err := filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return err
		}
		if _, expected := expectedFiles[relPath]; !expected {
			t.Errorf("unexpected file exists: %s", relPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking directory: %v", err)
	}
}

func TestCleanGeneratedFiles_NonExistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	ctx := &generator.Context{
		RootDir: tmpDir,
		Config: generator.Config{
			LogLevel: generator.LogLevelQuiet,
			DryRun:   false,
		},
	}

	// Should not error when directory doesn't exist
	if err := cleanGeneratedFiles(ctx, nonExistent); err != nil {
		t.Errorf("cleanGeneratedFiles should not error on non-existent directory: %v", err)
	}
}
