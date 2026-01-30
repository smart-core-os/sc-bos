package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// outputPackageTypeAnalysis outputs the package type analysis in the specified format
func outputPackageTypeAnalysis(analysis []PackageTypeAnalysis) error {
	var writer io.Writer = os.Stdout
	if *analyzeFile != "" {
		file, err := os.Create(*analyzeFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	switch *analyzeOutput {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{
			"analysis_type": "package-types",
			"results":       analysis,
		})
	case "csv":
		fmt.Fprintln(writer, "PackageType,Count,DependentRepos,Packages")
		for _, a := range analysis {
			fmt.Fprintf(writer, "%s,%d,%d,\"%s\"\n",
				a.PackageType, a.Count, a.DependentRepos, strings.Join(a.Packages, ";"))
		}
		return nil
	default: // text
		fmt.Fprintln(writer, "Package Type Analysis (by dependent repo location)")
		fmt.Fprintln(writer, "==================================================")
		fmt.Fprintln(writer)

		// Calculate total imports for percentage
		totalImports := 0
		for _, a := range analysis {
			totalImports += a.Count
		}

		for _, a := range analysis {
			percentage := 0.0
			if totalImports > 0 {
				percentage = float64(a.Count) / float64(totalImports) * 100
			}

			fmt.Fprintf(writer, "Import Location Type: %s\n", a.PackageType)
			fmt.Fprintf(writer, "  Import Count: %d (%.1f%%)\n", a.Count, percentage)
			fmt.Fprintf(writer, "  Dependent Repositories: %d\n", a.DependentRepos)
			fmt.Fprintf(writer, "  Unique Packages Imported: %d\n", len(a.Packages))

			// Show example files
			if len(a.ExampleFiles) > 0 {
				fmt.Fprintln(writer, "  Example files:")
				for _, file := range a.ExampleFiles {
					fmt.Fprintf(writer, "    - %s\n", file)
				}
			}

			if len(a.Packages) <= 10 {
				fmt.Fprintln(writer, "  Packages imported:")
				for _, pkg := range a.Packages {
					count := a.PackageCounts[pkg]
					fmt.Fprintf(writer, "    - %s (%d imports)\n", pkg, count)
				}
			} else {
				fmt.Fprintf(writer, "  Top 10 packages imported:\n")
				for i := 0; i < 10; i++ {
					count := a.PackageCounts[a.Packages[i]]
					fmt.Fprintf(writer, "    - %s (%d imports)\n", a.Packages[i], count)
				}
				fmt.Fprintf(writer, "    ... and %d more\n", len(a.Packages)-10)
			}
			fmt.Fprintln(writer)
		}
		return nil
	}
}

// outputRepoDependencyAnalysis outputs the repository dependency analysis in the specified format
func outputRepoDependencyAnalysis(analysis []RepoDependencyAnalysis) error {
	var writer io.Writer = os.Stdout
	if *analyzeFile != "" {
		file, err := os.Create(*analyzeFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	switch *analyzeOutput {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{
			"analysis_type": "repo-dependencies",
			"results":       analysis,
		})
	case "csv":
		fmt.Fprintln(writer, "Repository,PackageCount")
		for _, a := range analysis {
			fmt.Fprintf(writer, "%s,%d\n", a.Repository, a.PackageCount)
		}
		return nil
	default: // text
		fmt.Fprintln(writer, "Repository Dependency Analysis")
		fmt.Fprintln(writer, "=============================")
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "Repositories importing sc-bos packages:")
		fmt.Fprintln(writer)

		for i, a := range analysis {
			fmt.Fprintf(writer, "%d. %s - %d package(s)\n", i+1, a.Repository, a.PackageCount)
		}
		return nil
	}
}

// outputBasicSummary outputs a basic summary of the scan results
func outputBasicSummary(result *ScanResult) error {
	var writer io.Writer = os.Stdout
	if *analyzeFile != "" {
		file, err := os.Create(*analyzeFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	fmt.Fprintln(writer, "Scan Summary")
	fmt.Fprintln(writer, "============")
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "Base Module: %s\n", result.BaseModule)
	fmt.Fprintf(writer, "Organization: %s\n", result.Organization)
	fmt.Fprintf(writer, "Scan Date: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "Total Repositories: %d\n", result.TotalRepos)
	fmt.Fprintf(writer, "Repositories with Go Files: %d\n", result.ReposWithGoFiles)
	fmt.Fprintf(writer, "Repositories with Imports: %d\n", result.ReposWithImports)
	fmt.Fprintf(writer, "Unique Packages Imported: %d\n", len(result.PackageUsage))

	return nil
}
