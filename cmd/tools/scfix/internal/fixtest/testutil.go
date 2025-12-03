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
		AssertFileContent(t, tc.TestFile, tc.InputContent)
	})

	t.Run("Apply", func(t *testing.T) {
		tc := ReadTestCase(t, txtarPath)
		ctx := newContext(tc.TempDir, false, false)

		changes, err := runFunc(ctx)
		if err != nil {
			t.Fatalf("Fix failed: %v", err)
		}

		assertChangeCount(t, tc.ExpectedChanges, changes)
		AssertFileContent(t, tc.TestFile, tc.OutputContent)
	})
}

// TestCase represents a single test case from a txtar archive.
type TestCase struct {
	Archive         *txtar.Archive
	InputContent    []byte
	OutputContent   []byte
	ExpectedChanges int
	TempDir         string
	TestFile        string
}

// ReadTestCase loads and prepares a test case from a txtar file.
// The txtar file should contain:
// - input/test.ext: the input source code (any extension)
// - output/test.ext: the expected output source code (matching extension)
// - a comment with "expected_changes: N" indicating the expected number of changes.
func ReadTestCase(t *testing.T, txtarPath string) *TestCase {
	t.Helper()

	archive, err := txtar.ParseFile(txtarPath)
	if err != nil {
		t.Fatalf("Failed to parse txtar file: %v", err)
	}

	tempDir := t.TempDir()

	var inputContent, outputContent []byte
	var inputFile string

	// Find input and output files with any extension
	for _, file := range archive.Files {
		if strings.HasPrefix(file.Name, "input/") {
			inputContent = file.Data
			inputFile = file.Name
		} else if strings.HasPrefix(file.Name, "output/") {
			outputContent = file.Data
		}
	}

	if len(inputContent) == 0 {
		t.Fatal("input/test.* not found in txtar")
	}
	if len(outputContent) == 0 {
		t.Fatal("output/test.* not found in txtar")
	}

	// Extract filename from input path (preserving subdirectories)
	testFileName := strings.TrimPrefix(inputFile, "input/")
	testFile := filepath.Join(tempDir, testFileName)

	// Create parent directories if needed
	if dir := filepath.Dir(testFile); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create parent directories: %v", err)
		}
	}

	if err := os.WriteFile(testFile, inputContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	expectedChanges := parseExpectedChanges(t, string(archive.Comment))

	return &TestCase{
		Archive:         archive,
		InputContent:    inputContent,
		OutputContent:   outputContent,
		ExpectedChanges: expectedChanges,
		TempDir:         tempDir,
		TestFile:        testFile,
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

// AssertFileContent verifies output using line-by-line comparison with diff output.
func AssertFileContent(t *testing.T, testFile string, expectedContent []byte) {
	t.Helper()

	modified, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedLines := strings.Split(string(expectedContent), "\n")
	gotLines := strings.Split(string(modified), "\n")

	if diff := cmp.Diff(expectedLines, gotLines); diff != "" {
		t.Errorf("Output mismatch (-want +got):\n%s", diff)
	}
}
