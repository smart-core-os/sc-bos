// Package fixtest provides utilities for testing code fixers using txtar archives.
package fixtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/txtar"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

// RunDir runs tests for all txtar files in the specified directory.
// Each txtar file represents a separate test case.
// Argument dirPath is the path to the directory containing txtar files.
// Argument runFunc should run the fix and return the number of changes that were/would have been made.
func RunDir(t *testing.T, dirPath string, runFunc func(*fixer.Context) (int, error)) {
	t.Helper()

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("Failed to read directory %s: %v", dirPath, err)
	}

	if len(entries) == 0 {
		t.Fatalf("No test files found in directory %s", dirPath)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txtar") {
			continue
		}

		txtarPath := filepath.Join(dirPath, entry.Name())
		name := strings.TrimSuffix(entry.Name(), ".txtar")
		t.Run(name, func(t *testing.T) {
			Run(t, txtarPath, runFunc)
		})
	}
}

// Run runs tests to check the fixer correctly transforms the input into output from txtarPath.
// Argument txtarPath should refer to a file path of a txtar file, see [ReadTestCase] for details of contents.
// Argument runFunc should run the fix and return the number of changes that were/would have been made.
func Run(t *testing.T, txtarPath string, runFunc func(*fixer.Context) (int, error)) {
	t.Helper()

	t.Run("DryRun", func(t *testing.T) {
		tc := ReadTestCase(t, txtarPath)
		ctx := newContext(tc.TempDir, true, false)

		changes, err := runFunc(ctx)
		if err != nil {
			t.Fatalf("Fix failed: %v", err)
		}

		assertChangeCount(t, tc.ExpectedChanges, changes)

		// In dry-run mode, verify input files are unchanged
		AssertAllFilesMatch(t, tc.TempDir, tc.InputFiles)
	})

	t.Run("Apply", func(t *testing.T) {
		tc := ReadTestCase(t, txtarPath)
		ctx := newContext(tc.TempDir, false, false)

		changes, err := runFunc(ctx)
		if err != nil {
			t.Fatalf("Fix failed: %v", err)
		}

		assertChangeCount(t, tc.ExpectedChanges, changes)

		// Verify all output files exist with correct content
		AssertAllFilesMatch(t, tc.TempDir, tc.OutputFiles)
	})
}

// TestCase represents a single test case from a txtar archive.
type TestCase struct {
	Archive         *txtar.Archive
	InputFiles      map[string][]byte // Map of relative path to content for all input files
	OutputFiles     map[string][]byte // Map of relative path to content for all output files
	ExpectedChanges int
	TempDir         string
}

// ReadTestCase loads and prepares a test case from a txtar file.
// The txtar file should contain:
// - input/path/to/file: input files (can be multiple)
// - output/path/to/file: expected output files (can be multiple, may have different paths than input for file moves)
// - a comment with "expected_changes: N" indicating the expected number of changes.
func ReadTestCase(t *testing.T, txtarPath string) *TestCase {
	t.Helper()

	archive, err := txtar.ParseFile(txtarPath)
	if err != nil {
		t.Fatalf("Failed to parse txtar file: %v", err)
	}

	tempDir := t.TempDir()

	inputFiles := make(map[string][]byte)
	outputFiles := make(map[string][]byte)

	// Process all files in the archive
	for _, file := range archive.Files {
		if strings.HasPrefix(file.Name, "input/") {
			// Extract relative path from input/
			relPath := strings.TrimPrefix(file.Name, "input/")
			inputFiles[relPath] = file.Data

			// Write input file to temp directory
			fullPath := filepath.Join(tempDir, relPath)
			if dir := filepath.Dir(fullPath); dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create parent directories for %s: %v", relPath, err)
				}
			}
			if err := os.WriteFile(fullPath, file.Data, 0644); err != nil {
				t.Fatalf("Failed to write input file %s: %v", relPath, err)
			}
		} else if strings.HasPrefix(file.Name, "output/") {
			// Extract relative path from output/
			relPath := strings.TrimPrefix(file.Name, "output/")
			outputFiles[relPath] = file.Data
		}
	}

	if len(inputFiles) == 0 {
		t.Fatal("No input files found in txtar (files should start with 'input/')")
	}
	if len(outputFiles) == 0 {
		t.Fatal("No output files found in txtar (files should start with 'output/')")
	}

	expectedChanges := parseExpectedChanges(t, string(archive.Comment))

	return &TestCase{
		Archive:         archive,
		InputFiles:      inputFiles,
		OutputFiles:     outputFiles,
		ExpectedChanges: expectedChanges,
		TempDir:         tempDir,
	}
}

// parseExpectedChanges extracts the expected_changes value from the txtar comment.
func parseExpectedChanges(t *testing.T, comment string) int {
	t.Helper()

	lines := strings.Split(comment, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "expected_changes:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				var changes int
				if _, err := fmt.Sscanf(value, "%d", &changes); err == nil {
					return changes
				}
			}
		}
	}
	t.Fatal("expected_changes not found in txtar comment")
	return 0
}

// newContext creates a fixer.Context for testing.
func newContext(tempDir string, dryRun, verbose bool) *fixer.Context {
	return &fixer.Context{
		RootDir: tempDir,
		Config: fixer.Config{
			DryRun:      dryRun,
			VerboseMode: verbose,
		},
	}
}

// assertChangeCount verifies that the number of changes matches the expected count.
func assertChangeCount(t *testing.T, expected, got int) {
	t.Helper()
	if got != expected {
		t.Errorf("Expected %d changes, got %d", expected, got)
	}
}

// AssertAllFilesMatch verifies that the directory contains exactly the expected output files
// with the correct content. This is useful for fixers that move or remove files.
func AssertAllFilesMatch(t *testing.T, rootDir string, expectedFiles map[string][]byte) {
	t.Helper()

	// Collect all actual files in the directory
	actualFiles := make(map[string][]byte)
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		actualFiles[relPath] = content
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}

	// Check for unexpected files
	for relPath := range actualFiles {
		if _, expected := expectedFiles[relPath]; !expected {
			t.Errorf("Unexpected file exists: %s", relPath)
		}
	}

	// Check all expected files exist with correct content
	for relPath, expectedContent := range expectedFiles {
		actualContent, exists := actualFiles[relPath]
		if !exists {
			t.Errorf("Expected file missing: %s", relPath)
			continue
		}

		expectedLines := strings.Split(string(expectedContent), "\n")
		gotLines := strings.Split(string(actualContent), "\n")

		if diff := cmp.Diff(expectedLines, gotLines); diff != "" {
			t.Errorf("Content mismatch for %s (-want +got):\n%s", relPath, diff)
		}
	}
}
