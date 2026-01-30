package protov1

import (
	"regexp"
)

var (
	// Only match top-level messages/enums (no leading whitespace)
	messageRe = regexp.MustCompile(`(?m)^message\s+(\w+)\s*\{`)
	enumRe    = regexp.MustCompile(`(?m)^enum\s+(\w+)\s*\{`)
	// Capture both the import prefix and the path for rewriting
	importRe  = regexp.MustCompile(`(?m)^(\s*import\s+)"([^"]+\.proto)"\s*;`)
	packageRe = regexp.MustCompile(`(?m)^package\s+([\w.]+)\s*;`)
	serviceRe = regexp.MustCompile(`(?m)^service\s+(\w+)\s*\{`)
)

// extractPackageName extracts the package name from proto content.
// Returns empty string if no package declaration is found.
func extractPackageName(content []byte) string {
	matches := packageRe.FindSubmatch(content)
	if len(matches) > 1 {
		return string(matches[1])
	}
	return ""
}

// extractAllServices extracts all service names from proto content.
func extractAllServices(content []byte) []string {
	var services []string
	matches := serviceRe.FindAllSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			services = append(services, string(match[1]))
		}
	}
	return services
}

// extractTypeNames extracts all top-level message and enum names defined in a proto file.
func extractTypeNames(content []byte) []string {
	var types []string

	messageMatches := messageRe.FindAllSubmatch(content, -1)
	for _, match := range messageMatches {
		if len(match) > 1 {
			types = append(types, string(match[1]))
		}
	}

	enumMatches := enumRe.FindAllSubmatch(content, -1)
	for _, match := range enumMatches {
		if len(match) > 1 {
			types = append(types, string(match[1]))
		}
	}

	return types
}

// extractImports extracts all import paths from proto content.
func extractImports(content []byte) []string {
	var imports []string
	matches := importRe.FindAllSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 2 {
			imports = append(imports, string(match[2]))
		}
	}
	return imports
}
