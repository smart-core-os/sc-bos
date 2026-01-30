package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// FileUpdate represents a single line change in a file
type FileUpdate struct {
	path         string
	lineNumber   int
	originalLine string
	updatedLine  string
	replacement  string
}

// processFile scans a file and returns all necessary updates
func processFile(filePath string, presets []string, replacements []Replacement) ([]FileUpdate, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var updates []FileUpdate
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		updatedLine := line

		// Try each replacement pattern
		for _, repl := range replacements {
			// Skip this replacement if it doesn't apply to the current presets
			if !appliesToPreset(repl, presets) {
				continue
			}

			if repl.pattern.MatchString(updatedLine) {
				newLine := repl.pattern.ReplaceAllString(updatedLine, repl.repl)
				if newLine != updatedLine {
					updates = append(updates, FileUpdate{
						path:         filePath,
						lineNumber:   lineNumber,
						originalLine: line,
						updatedLine:  newLine,
						replacement:  repl.name,
					})
					updatedLine = newLine
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return updates, nil
}

// applyUpdates writes all updates to their respective files
func applyUpdates(updates map[string][]FileUpdate) error {
	for filePath, fileUpdates := range updates {
		if err := updateFile(filePath, fileUpdates); err != nil {
			return fmt.Errorf("failed to update %s: %w", filePath, err)
		}
		if *verbose {
			fmt.Printf("Updated %s\n", filePath)
		}
	}
	return nil
}

// updateFile applies updates to a single file
func updateFile(filePath string, updates []FileUpdate) error {
	// Read the entire file once
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Detect line ending style
	lineEnding := "\n"
	if strings.Contains(string(content), "\r\n") {
		lineEnding = "\r\n"
	}

	// Split into lines
	lines := strings.Split(string(content), lineEnding)

	// Apply updates (updates are 1-indexed)
	updateMap := make(map[int]string)
	for _, update := range updates {
		updateMap[update.lineNumber] = update.updatedLine
	}

	for i := range lines {
		if newLine, ok := updateMap[i+1]; ok {
			lines[i] = newLine
		}
	}

	// Write back to file
	output := strings.Join(lines, lineEnding)

	return os.WriteFile(filePath, []byte(output), 0644)
}

// isGeneratedFileWithReferences checks if a file is auto-generated and contains references
// This function reads the file only once to check both conditions efficiently
func isGeneratedFileWithReferences(filePath string, presets []string, replacements []Replacement) (isGenerated bool, hasReferences bool, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Check for generated file marker in first 10 lines
		if lineCount <= 10 && !isGenerated {
			if strings.Contains(line, "Code generated") && strings.Contains(line, "DO NOT EDIT") {
				isGenerated = true
			}
		}

		// Check for references (only if we've determined it's generated)
		if isGenerated && !hasReferences {
			for _, repl := range replacements {
				if !appliesToPreset(repl, presets) {
					continue
				}
				if repl.pattern.MatchString(line) {
					hasReferences = true
					break
				}
			}
		}

		// Early exit if we found both
		if isGenerated && hasReferences {
			return true, true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return isGenerated, hasReferences, err
	}

	return isGenerated, hasReferences, nil
}

// isGoSumFileWithReferences checks if a go.sum file contains references
// Returns (isGoSum, hasReferences, error)
func isGoSumFileWithReferences(filePath string, presets []string, replacements []Replacement) (isGoSum bool, hasReferences bool, err error) {
	// Check if it's a go.sum file by name
	if !strings.HasSuffix(filePath, "go.sum") {
		return false, false, nil
	}

	// It's a go.sum file, now check for references
	hasReferences, err = fileHasReferences(filePath, presets, replacements)
	return true, hasReferences, err
}

// fileHasReferences checks if a file contains any matching patterns
func fileHasReferences(filePath string, presets []string, replacements []Replacement) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if any replacement pattern matches
		for _, repl := range replacements {
			if !appliesToPreset(repl, presets) {
				continue
			}
			if repl.pattern.MatchString(line) {
				return true, nil
			}
		}
	}
	return false, scanner.Err()
}
