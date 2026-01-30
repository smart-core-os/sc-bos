package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// multiValue is a custom flag type that supports both comma-separated values and repeated flags
type multiValue []string

func (m *multiValue) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValue) Set(value string) error {
	// Split by comma and append each non-empty value
	for _, v := range strings.Split(value, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			*m = append(*m, v)
		}
	}
	return nil
}

// ProjectSpec holds information about a project and its target branch
type ProjectSpec struct {
	Name   string
	Branch string
}

// Default branches for known projects
var defaultBranches = map[string]string{
	"sc-bos":   "main",
	"gobacnet": "write",
}

var (
	// Flags
	dryRun       = flag.Bool("dry-run", false, "Show what would be changed without making changes")
	verbose      = flag.Bool("verbose", false, "Show detailed output")
	path         = flag.String("path", ".", "Path to scan for files")
	fileTypes    multiValue
	presets      multiValue
	projects     multiValue
	projectSpecs []ProjectSpec // Parsed project specifications
)

// Preset categories define which replacements to apply
var presetInfo = map[string]string{
	"all":    "All replacements (default)",
	"go":     "Go import path replacements only",
	"js":     "JavaScript/npm package replacements only",
	"docker": "Docker image replacements only",
	"docs":   "GitHub URL replacements only",
}

func init() {
	flag.Var(&fileTypes, "type", "File extensions to process (can be comma-separated or repeated)")
	flag.Var(&presets, "preset", "Preset category: all, go, js, docker, docs (can be comma-separated or repeated)")
	flag.Var(&projects, "project", "Projects to migrate (can be comma-separated or repeated, default: sc-bos,gobacnet)\n"+
		"\tFormat: project or project@branch (e.g., sc-bos@main, myproject@develop)")
}

func setupFlags() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Migrates references from vanti-dev to smart-core-os organization.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nPresets (control which replacements to apply):\n")
		// Generate preset list dynamically from presetInfo
		presetOrder := []string{"all", "go", "js", "docker", "docs"}
		for _, name := range presetOrder {
			if desc, ok := presetInfo[name]; ok {
				fmt.Fprintf(os.Stderr, "  %-6s - %s\n", name, desc)
			}
		}
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --dry-run --preset go --path ~/my-project\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --preset js --path ~/my-project\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --preset docker --verbose --path ~/my-project\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --type go,mod,yaml --path .\n", os.Args[0])
	}
}

func parseAndValidateFlags() error {
	flag.Parse()

	// Set default values if not provided
	if len(projects) == 0 {
		projects = multiValue{"sc-bos", "gobacnet"}
	}
	if len(presets) == 0 {
		presets = multiValue{"all"}
	}

	// Validate presets
	for _, p := range presets {
		if _, ok := presetInfo[p]; !ok {
			return fmt.Errorf("unknown preset: %s (available: all, go, js, docker, docs)", p)
		}
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects specified")
	}

	// Parse project specifications
	projectSpecs = parseProjectSpecs(projects)

	return nil
}

// parseProjectSpecs parses project specifications which can be in format "project" or "project@branch"
func parseProjectSpecs(projects []string) []ProjectSpec {
	specs := make([]ProjectSpec, 0, len(projects))
	for _, project := range projects {
		parts := strings.SplitN(project, "@", 2)
		spec := ProjectSpec{
			Name: parts[0],
		}
		if len(parts) == 2 {
			// User specified a branch
			spec.Branch = parts[1]
		} else {
			// Use default branch for known projects, otherwise leave empty (go get will use @latest)
			if branch, ok := defaultBranches[spec.Name]; ok {
				spec.Branch = branch
			} else {
				// Leave empty - go get defaults to @latest
				spec.Branch = ""
			}
		}
		specs = append(specs, spec)
	}
	return specs
}

func parseExtensions(typesStr string) map[string]bool {
	extensions := make(map[string]bool)
	for _, ext := range strings.Split(typesStr, ",") {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			extensions[ext] = true
		}
	}
	return extensions
}
