package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hasGoModDependencies checks if go.mod has dependencies on any of the projects being migrated
func hasGoModDependencies(rootPath string, projectSpecs []ProjectSpec) (bool, []ProjectSpec, error) {
	goModPath := filepath.Join(rootPath, "go.mod")

	// Check if go.mod exists
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return false, nil, nil
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return false, nil, err
	}
	defer file.Close()

	var foundDeps []ProjectSpec
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		for _, spec := range projectSpecs {
			oldPath := fmt.Sprintf("github.com/vanti-dev/%s", spec.Name)
			if strings.Contains(line, oldPath) {
				if !containsProject(foundDeps, spec.Name) {
					foundDeps = append(foundDeps, spec)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return false, nil, err
	}

	return len(foundDeps) > 0, foundDeps, nil
}

// removeDependencies removes specified dependencies from go.mod
func removeDependencies(rootPath string, projectSpecs []ProjectSpec) error {
	// Use go get with @none to cleanly remove dependencies
	for _, spec := range projectSpecs {
		oldPath := fmt.Sprintf("github.com/vanti-dev/%s@none", spec.Name)
		cmd := exec.Command("go", "get", oldPath)
		cmd.Dir = rootPath
		if output, err := cmd.CombinedOutput(); err != nil {
			// If @none fails, try using go mod edit as fallback
			oldPathNoVersion := fmt.Sprintf("github.com/vanti-dev/%s", spec.Name)
			cmd = exec.Command("go", "mod", "edit", "-droprequire", oldPathNoVersion)
			cmd.Dir = rootPath
			if output2, err2 := cmd.CombinedOutput(); err2 != nil {
				return fmt.Errorf("failed to remove dependency %s: %w\n%s\n%s", spec.Name, err, output, output2)
			}
		}
	}

	return nil
}

// addDependencies adds dependencies using the new organization with project-specific branches
func addDependencies(rootPath string, projectSpecs []ProjectSpec) error {
	// First remove any migrated references that might exist from the text replacement
	for _, spec := range projectSpecs {
		newPath := fmt.Sprintf("github.com/smart-core-os/%s", spec.Name)
		cmd := exec.Command("go", "mod", "edit", "-droprequire", newPath)
		cmd.Dir = rootPath
		// Ignore errors - the dependency might not exist
		cmd.CombinedOutput()
	}

	// Now add the dependencies from the new organization using project-specific branches
	for _, spec := range projectSpecs {
		var newPath string
		if spec.Branch != "" {
			newPath = fmt.Sprintf("github.com/smart-core-os/%s@%s", spec.Name, spec.Branch)
		} else {
			// No branch specified - go get will use @latest
			newPath = fmt.Sprintf("github.com/smart-core-os/%s", spec.Name)
		}
		cmd := exec.Command("go", "get", newPath)
		cmd.Dir = rootPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to add dependency %s: %w\n%s", newPath, err, output)
		}
	}
	return nil
}

// runGoModTidy runs go mod tidy in the specified directory
func runGoModTidy(rootPath string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = rootPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\n%s", err, output)
	}
	return nil
}

// containsProject checks if a ProjectSpec slice contains a project by name
func containsProject(specs []ProjectSpec, name string) bool {
	for _, spec := range specs {
		if spec.Name == name {
			return true
		}
	}
	return false
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
