package main

import (
	"fmt"
	"regexp"
)

// Replacement represents a pattern replacement with preset restrictions
type Replacement struct {
	name    string
	pattern *regexp.Regexp
	repl    string
	presets []string // Which presets this replacement applies to (empty means all)
}

// buildReplacements creates replacement patterns for the given projects
func buildReplacements(projectList []string) []Replacement {
	var replacements []Replacement

	for _, project := range projectList {
		// GHCR image path
		replacements = append(replacements, Replacement{
			name:    fmt.Sprintf("GHCR image path (%s)", project),
			pattern: regexp.MustCompile(fmt.Sprintf(`ghcr\.io/vanti-dev/%s`, regexp.QuoteMeta(project))),
			repl:    fmt.Sprintf("ghcr.io/smart-core-os/%s", project),
			presets: []string{"docker", "all"},
		})

		// GitHub raw URL
		replacements = append(replacements, Replacement{
			name:    fmt.Sprintf("GitHub raw URL (%s)", project),
			pattern: regexp.MustCompile(fmt.Sprintf(`https?://raw\.githubusercontent\.com/vanti-dev/%s`, regexp.QuoteMeta(project))),
			repl:    fmt.Sprintf("https://raw.githubusercontent.com/smart-core-os/%s", project),
			presets: []string{"docs", "all"},
		})

		// GitHub URL
		replacements = append(replacements, Replacement{
			name:    fmt.Sprintf("GitHub URL (%s)", project),
			pattern: regexp.MustCompile(fmt.Sprintf(`https?://github\.com/vanti-dev/%s`, regexp.QuoteMeta(project))),
			repl:    fmt.Sprintf("https://github.com/smart-core-os/%s", project),
			presets: []string{"docs", "all"},
		})

		// Go import path
		replacements = append(replacements, Replacement{
			name:    fmt.Sprintf("Go import path (%s)", project),
			pattern: regexp.MustCompile(fmt.Sprintf(`(^|[^:/])github\.com/vanti-dev/%s`, regexp.QuoteMeta(project))),
			repl:    fmt.Sprintf("${1}github.com/smart-core-os/%s", project),
			presets: []string{"go", "all"},
		})

		// npm package (only for sc-bos which has npm packages)
		if project == "sc-bos" {
			replacements = append(replacements, Replacement{
				name:    "npm package",
				pattern: regexp.MustCompile(`@vanti-dev/sc-bos`),
				repl:    "@smart-core-os/sc-bos",
				presets: []string{"js", "all"},
			})
		}
	}

	return replacements
}

// appliesToPreset checks if a replacement applies to any of the given presets
func appliesToPreset(repl Replacement, presets []string) bool {
	// Empty presets list means applies to all
	if len(repl.presets) == 0 {
		return true
	}
	// Check if any of the current presets is in the replacement's preset list
	for _, p := range presets {
		for _, rp := range repl.presets {
			if rp == p {
				return true
			}
		}
	}
	return false
}
