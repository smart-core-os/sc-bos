package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// logResults shows the changes that were found
func logResults(updates map[string][]FileUpdate) {
	if len(updates) == 0 {
		fmt.Println("No references to vanti-dev found.")
		return
	}

	fmt.Printf("\nFound references in %d file(s):\n\n", len(updates))

	// Get sorted file list using stdlib
	files := slices.Sorted(maps.Keys(updates))

	totalUpdates := 0
	for _, file := range files {
		fileUpdates := updates[file]
		totalUpdates += len(fileUpdates)
		fmt.Printf("%s: (%d change(s))\n", file, len(fileUpdates))
		if *verbose {
			for _, update := range fileUpdates {
				fmt.Printf("  Line %d [%s]:\n", update.lineNumber, update.replacement)
				fmt.Printf("    - %s\n", strings.TrimSpace(update.originalLine))
				fmt.Printf("    + %s\n", strings.TrimSpace(update.updatedLine))
			}
		}
	}

	fmt.Printf("\nTotal: %d change(s) in %d file(s)\n", totalUpdates, len(updates))
}

// logRenamedFiles shows renamed files
func logRenamedFiles(renamedFiles map[string]string) {
	if len(renamedFiles) == 0 {
		return
	}

	fmt.Printf("\n✓ Renamed %d file(s) with vanti-dev in their names:\n", len(renamedFiles))

	// Sort old names for consistent output using stdlib
	oldNames := slices.Sorted(maps.Keys(renamedFiles))
	for _, oldName := range oldNames {
		fmt.Printf("  %s -> %s\n", oldName, renamedFiles[oldName])
	}
}

// logGoSumWarning shows a warning about skipped go.sum files
func logGoSumWarning(goSumWithRefs []string) {
	slices.Sort(goSumWithRefs)
	fmt.Fprintf(os.Stderr, "\nWarning: %d go.sum file(s) contain references but were skipped:\n", len(goSumWithRefs))
	for _, file := range goSumWithRefs {
		fmt.Fprintf(os.Stderr, "  - %s\n", file)
	}

	fmt.Fprintf(os.Stderr, "\nYou need to regenerate these files after updating go.mod.\n")
	fmt.Fprintf(os.Stderr, "Run: go mod tidy\n")
}

// logGeneratedFilesWarning shows a warning about skipped generated files
func logGeneratedFilesWarning(generatedWithRefs []string) {
	// If there are more than 5 files, just show unique directories
	if len(generatedWithRefs) > 5 {
		fmt.Fprintf(os.Stderr, "\nWarning: %d generated files in the following directories contain references but were skipped:\n", len(generatedWithRefs))
		logUniqueDirectories(generatedWithRefs, os.Stderr)
	} else {
		slices.Sort(generatedWithRefs)
		fmt.Fprintf(os.Stderr, "\nWarning: %d generated file(s) contain references but were skipped:\n", len(generatedWithRefs))
		for _, file := range generatedWithRefs {
			fmt.Fprintf(os.Stderr, "  - %s\n", file)
		}
	}

	fmt.Fprintf(os.Stderr, "\nYou may need to regenerate these files after updating their source files.\n")
	fmt.Fprintf(os.Stderr, "Look for //go:generate directives or run: go generate ./...\n")
}

// logUniqueDirectories extracts and displays unique directories containing the files
func logUniqueDirectories(files []string, output *os.File) {
	// Group files by directory to count them
	dirCounts := make(map[string]int)
	for _, file := range files {
		dir := filepath.Dir(file)
		dirCounts[dir]++
	}

	// Get sorted directory list using stdlib
	dirs := slices.Sorted(maps.Keys(dirCounts))

	// Display directories with counts
	for _, dir := range dirs {
		count := dirCounts[dir]
		fmt.Fprintf(output, "  - %s/ (%d file(s))\n", dir, count)
	}

	if *verbose {
		fmt.Fprintf(output, "\nIndividual files:\n")
		// Sort files for consistent output
		sortedFiles := slices.Clone(files)
		slices.Sort(sortedFiles)
		for _, file := range sortedFiles {
			fmt.Fprintf(output, "  - %s\n", file)
		}
	} else {
		fmt.Fprintf(output, "  (run with --verbose to see individual files)\n")
	}
}
