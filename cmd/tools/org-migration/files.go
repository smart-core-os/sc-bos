package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Default file extensions to scan when --types is not specified
const defaultFileTypes = "go,mod,sum,proto,sh,yml,yaml,json,md,js,ts,jsx,tsx,mjs,cjs,vue,html,xml,Dockerfile"

// collectFiles recursively collects files with specified extensions
func collectFiles(rootPath string, extensions map[string]bool) ([]string, error) {
	var files []string

	// Directories to skip during traversal
	// Note: .run and .github are NOT skipped to allow updating IDEA run configurations and GitHub workflows
	ignoredDirs := []string{"node_modules", "vendor", "dist", "org-migration"}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common ignored directories
		if d.IsDir() {
			name := d.Name()
			// Don't skip the root directory even if it's "."
			if path != rootPath {
				// Allow .run directory (IDEA run configurations) and .github directory (GitHub workflows)
				if name == ".run" || name == ".github" {
					return nil
				}
				// Skip other hidden directories (starting with .)
				if strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
				// Skip explicitly ignored directories
				for _, ignored := range ignoredDirs {
					if name == ignored {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		// Check if file extension matches
		name := d.Name()
		// Handle Dockerfile specially (no extension)
		if name == "Dockerfile" || strings.HasPrefix(name, "Dockerfile-") {
			files = append(files, path)
			return nil
		}

		// Check regular extensions
		ext := strings.TrimPrefix(filepath.Ext(name), ".")
		if ext != "" && extensions[ext] {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// renameFiles renames files that have vanti-dev in their names for the specified projects
func renameFiles(files []string, projects []string) (map[string]string, error) {
	renamedFiles := make(map[string]string)

	for _, filePath := range files {
		fileName := filepath.Base(filePath)

		// Check if filename contains vanti-dev and any of the selected projects
		if !strings.Contains(fileName, "vanti-dev") {
			continue
		}

		// Check if the filename contains any of the selected projects
		hasProject := false
		for _, project := range projects {
			if strings.Contains(fileName, project) {
				hasProject = true
				break
			}
		}

		if !hasProject {
			continue
		}

		// Create new filename by replacing vanti-dev with smart-core-os
		newFileName := strings.ReplaceAll(fileName, "vanti-dev", "smart-core-os")
		newFilePath := filepath.Join(filepath.Dir(filePath), newFileName)

		// Rename the file
		if err := os.Rename(filePath, newFilePath); err != nil {
			return renamedFiles, err
		}

		renamedFiles[filePath] = newFilePath
		if *verbose {
			// verbose is imported from flags.go
			// This is OK since they're in the same package
		}
	}

	return renamedFiles, nil
}
