// Command org-migration helps migrate references from vanti-dev to smart-core-os organization.
// This tool scans local files and updates references to the old organization name with the new one.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Setup and parse flags
	setupFlags()
	if err := parseAndValidateFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine which file types to process
	var typesStr string
	if len(fileTypes) > 0 {
		typesStr = strings.Join(fileTypes, ",")
	} else {
		typesStr = defaultFileTypes
	}
	extensions := parseExtensions(typesStr)

	// Build replacements for the specified projects
	// Extract just the project names from specs
	projectNames := make([]string, len(projectSpecs))
	for i, spec := range projectSpecs {
		projectNames[i] = spec.Name
	}
	replacements := buildReplacements(projectNames)

	// Collect files to process
	files, err := collectFiles(*path, extensions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting files: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Scanning %d files in %s\n", len(files), *path)
	}

	// Process files
	updates := make(map[string][]FileUpdate)
	var skippedGeneratedWithRefs []string
	var skippedGoSumWithRefs []string
	var skippedGoModFiles []string

	for _, file := range files {
		// Skip go.sum files - these should be regenerated with `go mod tidy`
		isGoSum, hasReferences, err := isGoSumFileWithReferences(file, presets, replacements)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking if %s is go.sum: %v\n", file, err)
			continue
		}
		if isGoSum && hasReferences {
			skippedGoSumWithRefs = append(skippedGoSumWithRefs, file)
			continue
		}
		if isGoSum {
			continue
		}

		// Skip generated files
		isGenerated, hasReferences, err := isGeneratedFileWithReferences(file, presets, replacements)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking if %s is generated: %v\n", file, err)
			continue
		}
		if isGenerated && hasReferences {
			skippedGeneratedWithRefs = append(skippedGeneratedWithRefs, file)
			continue
		}
		if isGenerated {
			continue
		}

		fileUpdates, err := processFile(file, presets, replacements)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", file, err)
			continue
		}
		if len(fileUpdates) > 0 {
			updates[file] = fileUpdates
		}
	}

	// If there are generated files with references, skip go.mod files to avoid broken state
	// Also track if we'll be doing automatic go.mod handling
	var willHandleGoMod bool
	if appliesToPreset(Replacement{presets: []string{"go", "all"}}, presets) {
		if hasDeps, _, _ := hasGoModDependencies(*path, projectSpecs); hasDeps && len(skippedGeneratedWithRefs) == 0 {
			willHandleGoMod = true
		}
	}

	if len(skippedGeneratedWithRefs) > 0 || willHandleGoMod {
		// Remove any go.mod files from updates
		for file := range updates {
			if strings.HasSuffix(file, "go.mod") {
				if len(skippedGeneratedWithRefs) > 0 {
					skippedGoModFiles = append(skippedGoModFiles, file)
				}
				delete(updates, file)
			}
		}
	}

	// Display results
	logResults(updates)

	// Warn about skipped files that contain references
	// Skip go.sum warning if we're handling go.mod automatically (go mod tidy will regenerate it)
	if len(skippedGoSumWithRefs) > 0 && !willHandleGoMod {
		logGoSumWarning(skippedGoSumWithRefs)
	}
	if len(skippedGeneratedWithRefs) > 0 {
		logGeneratedFilesWarning(skippedGeneratedWithRefs)
	}
	if len(skippedGoModFiles) > 0 {
		fmt.Fprintf(os.Stderr, "\nNote: Skipped updating go.mod files because generated files need to be regenerated first.\n")
		fmt.Fprintf(os.Stderr, "After regenerating, re-run this tool to update go.mod.\n")
	}

	// Exit early if no changes or dry-run
	if len(updates) == 0 {
		return
	}

	if *dryRun {
		fmt.Println("\n[DRY RUN] No changes were made. Run without --dry-run to apply changes.")

		// Show go.mod handling plan if applicable
		if appliesToPreset(Replacement{presets: []string{"go", "all"}}, presets) {
			hasDeps, deps, err := hasGoModDependencies(*path, projectSpecs)
			if err == nil && hasDeps {
				if len(skippedGeneratedWithRefs) > 0 {
					fmt.Printf("\nNote: go.mod will not be updated because generated files need regeneration first.\n")
					fmt.Printf("After regenerating files, re-run this tool to automatically update go.mod.\n")
				} else {
					fmt.Printf("\nNote: Will automatically update go.mod dependencies:\n")
					for _, dep := range deps {
						if dep.Branch != "" {
							fmt.Printf("  - %s (using @%s)\n", dep.Name, dep.Branch)
						} else {
							fmt.Printf("  - %s (using @latest)\n", dep.Name)
						}
					}
				}
			}
		}
		return
	}

	// Handle go.mod dependencies if targeting go files
	var needsGoModFix bool
	var goModDeps []ProjectSpec
	if appliesToPreset(Replacement{presets: []string{"go", "all"}}, presets) {
		hasDeps, deps, err := hasGoModDependencies(*path, projectSpecs)
		if err == nil && hasDeps && len(skippedGeneratedWithRefs) == 0 {
			needsGoModFix = true
			goModDeps = deps

			fmt.Printf("\nUpdating go.mod dependencies:\n")
			for _, dep := range deps {
				if dep.Branch != "" {
					fmt.Printf("  - %s (using @%s)\n", dep.Name, dep.Branch)
				} else {
					fmt.Printf("  - %s (using @latest)\n", dep.Name)
				}
			}

			if err := removeDependencies(*path, goModDeps); err != nil {
				fmt.Fprintf(os.Stderr, "Error removing dependencies: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// Apply updates
	if err := applyUpdates(updates); err != nil {
		fmt.Fprintf(os.Stderr, "Error applying updates: %v\n", err)
		os.Exit(1)
	}

	// Re-add go.mod dependencies if we removed them
	if needsGoModFix {
		if err := addDependencies(*path, goModDeps); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding dependencies: %v\n", err)
			fmt.Fprintf(os.Stderr, "You may need to manually run: go get github.com/smart-core-os/<project>@main\n")
			os.Exit(1)
		}

		if err := runGoModTidy(*path); err != nil {
			fmt.Fprintf(os.Stderr, "Error running go mod tidy: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✓ Go dependencies updated")
	}

	// Rename files that have vanti-dev in their names for the selected projects
	renamedFiles, err := renameFiles(files, projectNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error renaming files: %v\n", err)
		os.Exit(1)
	}
	logRenamedFiles(renamedFiles)

	fmt.Println("\n✓ Successfully updated all files.")
}
