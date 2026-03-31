package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// extractRepoNameFromModule extracts the repository owner/name from a Go module path
// For example: "github.com/smart-core-os/sc-bos" -> "smart-core-os/sc-bos"
func extractRepoNameFromModule(modulePath string) string {
	// Remove common prefixes and extract owner/repo
	parts := strings.Split(modulePath, "/")
	if len(parts) < 3 {
		return ""
	}

	// Handle github.com/owner/repo format
	if parts[0] == "github.com" && len(parts) >= 3 {
		return parts[1] + "/" + parts[2]
	}

	return ""
}

// extractGoFilesFromTree extracts all Go file paths from a Git Tree
// If moduleDirs is not nil, only includes files that are within those module directories
func extractGoFilesFromTree(tree *GitTree, moduleDirs []string) []string {
	var goFiles []string
	for _, item := range tree.Tree {
		// Only process files (blobs), not directories (trees)
		if item.Type != "blob" {
			continue
		}

		// Check if it's a Go file
		if !strings.HasSuffix(item.Path, ".go") {
			continue
		}

		// Skip files in directories we typically want to ignore
		if shouldSkipPath(item.Path) {
			continue
		}

		// If moduleDirs is specified, only include files within those directories
		if moduleDirs != nil {
			if !isInModuleDirs(item.Path, moduleDirs) {
				continue
			}
		}

		goFiles = append(goFiles, item.Path)
	}
	return goFiles
}

// isInModuleDirs checks if a file path is within any of the specified module directories
func isInModuleDirs(filePath string, moduleDirs []string) bool {
	for _, moduleDir := range moduleDirs {
		if moduleDir == "" {
			// Root directory - all files are included
			return true
		}
		// Check if file is in this module directory
		if strings.HasPrefix(filePath, moduleDir+"/") {
			return true
		}
	}
	return false
}

// shouldSkipPath checks if a file path should be skipped based on directory patterns
func shouldSkipPath(path string) bool {
	skipDirs := []string{
		".git/", ".github/", "vendor/", "node_modules/",
		".vscode/", ".idea/", "dist/", "build/", "target/",
		"docs/", "documentation/", "examples/", "test-data/",
		"ui/", "manifests/", "deploy/", "scripts/",
	}

	for _, skip := range skipDirs {
		if strings.HasPrefix(path, skip) || strings.Contains(path, "/"+skip) {
			return true
		}
	}
	return false
}

// classifyPackage determines the package type based on its path
func classifyPackage(pkg, baseModule string) string {
	// Remove base module prefix
	subPath := strings.TrimPrefix(pkg, baseModule+"/")
	if subPath == pkg {
		// Package is the base module itself
		return "core"
	}

	parts := strings.Split(subPath, "/")
	if len(parts) == 0 {
		return "unknown"
	}

	firstPart := parts[0]

	// Classify based on top-level directory
	switch {
	case firstPart == "cmd":
		if len(parts) >= 3 && parts[1] == "tools" {
			return "tools"
		}
		return "commands"
	case firstPart == "pkg":
		if len(parts) >= 2 {
			switch parts[1] {
			case "auto":
				return "automations"
			case "driver":
				return "drivers"
			case "gentrait":
				return "traits"
			case "app":
				return "app"
			case "node":
				return "node"
			}
		}
		return "packages"
	case firstPart == "internal":
		return "internal"
	case firstPart == "proto":
		return "proto"
	default:
		return firstPart
	}
}

// classifyImportByFilePath determines the category based on the file path in the dependent repository
// This classifies where the import is being used (e.g., in a tool, driver, automation, etc.)
func classifyImportByFilePath(filePath string) string {
	parts := strings.Split(filePath, "/")
	if len(parts) == 0 {
		return "other"
	}

	firstPart := parts[0]

	// Classify based on directory structure in the dependent repo
	switch {
	case firstPart == "cmd":
		if len(parts) >= 2 {
			if parts[1] == "tools" {
				return "tools"
			}
			return "commands"
		}
		return "commands"

	case firstPart == "internal":
		// Look deeper into internal to find specific types
		if len(parts) >= 2 {
			switch parts[1] {
			case "auto", "automations", "automation":
				return "automations"
			case "driver", "drivers":
				return "drivers"
			}
		}
		// Other internal code
		return "other-internal"

	case firstPart == "pkg":
		if len(parts) >= 2 {
			switch parts[1] {
			case "auto", "automations", "automation":
				return "automations"
			case "driver", "drivers":
				return "drivers"
			case "gentrait", "trait", "traits":
				return "traits"
			case "app":
				return "app"
			case "node":
				return "node"
			case "gen":
				return "generated"
			}
		}
		return "other-pkg"

	case firstPart == "proto":
		return "proto"
	case firstPart == "test", firstPart == "tests":
		return "tests"
	case firstPart == "example", firstPart == "examples":
		return "examples"
	default:
		// Root level or other
		return "other"
	}
}

// saveScanResult saves scan results to a JSON file
func saveScanResult(result ScanResult, filename string) error {
	return saveJSON(filename, result)
}

// loadScanResult loads scan results from a JSON file
func loadScanResult(filename string) (*ScanResult, error) {
	var result ScanResult
	if err := loadJSON(filename, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// saveJSON saves any data structure to a JSON file
func saveJSON(filename string, data any) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// loadJSON loads any data structure from a JSON file
func loadJSON(filename string, target any) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse input file: %v", err)
	}

	return nil
}
