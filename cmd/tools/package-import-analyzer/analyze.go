package main

import (
	"flag"
	"log"
	"sort"
)

// runAnalyze executes the analyze command
func runAnalyze() {
	if *analyzeInput == "" {
		log.Fatal("Input file is required. Use -input flag.")
	}

	result, err := loadScanResult(*analyzeInput)
	if err != nil {
		log.Fatalf("Failed to load scan result: %v", err)
	}

	if *verbose {
		log.Printf("Loaded scan results from %s", *analyzeInput)
		log.Printf("Base module: %s, Organization: %s", result.BaseModule, result.Organization)
		log.Printf("Scanned on: %s", result.Timestamp.Format("2006-01-02 15:04:05"))
	}

	switch *analyzeType {
	case "all":
		runAllAnalysis(result)
	case "package-types":
		runPackageTypeAnalysis(result)
	case "repo-dependencies":
		runRepoDependenciesAnalysis(result)
	default:
		log.Fatalf("Unknown analysis type: %s", *analyzeType)
	}
}

func runAllAnalysis(result *ScanResult) {
	runPackageTypeAnalysis(result)
	println()
	runRepoDependenciesAnalysis(result)
	println()
	runBasicSummary(result)
}

func runPackageTypeAnalysis(result *ScanResult) {
	analysis := analyzePackageTypes(result)

	if err := outputPackageTypeAnalysis(analysis); err != nil {
		log.Fatalf("Failed to output analysis: %v", err)
	}
}

func runRepoDependenciesAnalysis(result *ScanResult) {
	analysis := analyzeRepoDependencies(result)

	if err := outputRepoDependencyAnalysis(analysis); err != nil {
		log.Fatalf("Failed to output analysis: %v", err)
	}
}

func runBasicSummary(result *ScanResult) {
	if err := outputBasicSummary(result); err != nil {
		log.Fatalf("Failed to output summary: %v", err)
	}
}

func analyzePackageTypes(result *ScanResult) []PackageTypeAnalysis {
	// Map of type -> list of import locations (repo:file pairs)
	typeImports := make(map[string]map[string]bool)      // type -> unique "repo:file" strings
	typeDependents := make(map[string]map[string]bool)   // type -> dependent repos
	typePackages := make(map[string]map[string]bool)     // type -> packages being imported
	typeFiles := make(map[string][]string)               // type -> list of all files (for examples)
	typePackageCounts := make(map[string]map[string]int) // type -> package -> import count

	// Iterate through all package usage
	for pkg, importInfos := range result.PackageUsage {
		for _, info := range importInfos {
			// Classify each file based on where it's located in the dependent repo
			for _, filePath := range info.Files {
				fileType := classifyImportByFilePath(filePath)

				// Initialize maps if needed
				if typeImports[fileType] == nil {
					typeImports[fileType] = make(map[string]bool)
					typeDependents[fileType] = make(map[string]bool)
					typePackages[fileType] = make(map[string]bool)
					typeFiles[fileType] = make([]string, 0)
					typePackageCounts[fileType] = make(map[string]int)
				}

				// Track the import location
				importLocation := info.Repository + ":" + filePath
				typeImports[fileType][importLocation] = true

				// Track the dependent repository
				typeDependents[fileType][info.Repository] = true

				// Track the package being imported
				typePackages[fileType][pkg] = true

				// Count usage of this package in this type
				typePackageCounts[fileType][pkg]++

				// Collect file for examples (with repo prefix)
				typeFiles[fileType] = append(typeFiles[fileType], importLocation)
			}
		}
	}

	// Build the analysis results
	var analysis []PackageTypeAnalysis
	for fileType, imports := range typeImports {
		// Get packages sorted by usage count (most used first)
		type pkgCount struct {
			pkg   string
			count int
		}

		pkgCounts := make([]pkgCount, 0, len(typePackages[fileType]))
		for pkg := range typePackages[fileType] {
			pkgCounts = append(pkgCounts, pkgCount{
				pkg:   pkg,
				count: typePackageCounts[fileType][pkg],
			})
		}

		// Sort by count descending, then by name for ties
		sort.Slice(pkgCounts, func(i, j int) bool {
			if pkgCounts[i].count != pkgCounts[j].count {
				return pkgCounts[i].count > pkgCounts[j].count
			}
			return pkgCounts[i].pkg < pkgCounts[j].pkg
		})

		// Extract just the package names
		pkgList := make([]string, len(pkgCounts))
		for i, pc := range pkgCounts {
			pkgList[i] = pc.pkg
		}

		// Get example files (up to 5 unique examples)
		exampleFiles := make([]string, 0)
		fileList := typeFiles[fileType]
		// Remove duplicates and take up to 5
		seen := make(map[string]bool)
		for _, f := range fileList {
			if !seen[f] && len(exampleFiles) < 5 {
				exampleFiles = append(exampleFiles, f)
				seen[f] = true
			}
			if len(exampleFiles) >= 5 {
				break
			}
		}

		analysis = append(analysis, PackageTypeAnalysis{
			PackageType:    fileType,
			Count:          len(imports), // Count of unique file locations
			Packages:       pkgList,
			PackageCounts:  typePackageCounts[fileType],
			DependentRepos: len(typeDependents[fileType]),
			ExampleFiles:   exampleFiles,
		})
	}

	// Sort by count (descending)
	sort.Slice(analysis, func(i, j int) bool {
		return analysis[i].Count > analysis[j].Count
	})

	return analysis
}

func analyzeRepoDependencies(result *ScanResult) []RepoDependencyAnalysis {
	// Group by importing repository
	repoPackages := make(map[string]map[string]bool) // repo -> packages

	for pkg, importInfos := range result.PackageUsage {
		for _, info := range importInfos {
			if repoPackages[info.Repository] == nil {
				repoPackages[info.Repository] = make(map[string]bool)
			}
			repoPackages[info.Repository][pkg] = true
		}
	}

	var analysis []RepoDependencyAnalysis
	for repo, packages := range repoPackages {
		analysis = append(analysis, RepoDependencyAnalysis{
			Repository:   repo,
			PackageCount: len(packages),
		})
	}

	sort.Slice(analysis, func(i, j int) bool {
		return analysis[i].PackageCount > analysis[j].PackageCount
	})

	return analysis
}

var (
	// Analyze subcommand flags
	analyzeCmd    = flag.NewFlagSet("analyze", flag.ExitOnError)
	analyzeInput  = analyzeCmd.String("input", "", "Input file path with scan results (required)")
	analyzeOutput = analyzeCmd.String("output", "text", "Output format: json, csv, or text")
	analyzeFile   = analyzeCmd.String("output-file", "", "Output file path (if not specified, outputs to stdout)")
	analyzeType   = analyzeCmd.String("type", "all", "Analysis type: all, package-types, repo-dependencies")
)
