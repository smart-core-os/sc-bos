package historyimports

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

// relPath returns the path relative to the repository root, or the base filename if it can't be computed
func relPath(rootDir, absPath string) string {
	if rel, err := filepath.Rel(rootDir, absPath); err == nil {
		return rel
	}
	return filepath.Base(absPath)
}

// processSingleLineImport handles single-line import statements.
func processSingleLineImport(ctx *fixer.Context, filename, line, fileTraitName string) (string, bool) {
	matches := singleLineImportPattern.FindStringSubmatch(line)
	if matches == nil {
		return line, false
	}

	// matches: [full, prefix, symbols, middle, traitName, pbSuffix, extension]
	prefix := matches[1]     // "import {"
	symbolsStr := matches[2] // "Foo, Bar, Baz"
	middle := matches[3]     // "} from '@smart-core-os/sc-bos-ui-gen/proto/"
	traitName := matches[4]  // "transport" or "history" or "electric_history"
	pbSuffix := matches[5]   // "_grpc_web_pb" or "_pb"
	extension := matches[6]  // ".js';" or "';" or "\";"

	// Skip if already a trait-specific history import (ends with _history)
	if strings.HasSuffix(traitName, "_history") {
		return line, false
	}

	// Parse imported symbols
	symbols := parseSymbols(symbolsStr)

	// Special case: if importing from generic "history" file, all symbols are history-related
	// and we use the trait name determined from the HistoryClient in the file
	if traitName == "history" {
		// All symbols from history file are history symbols
		if len(symbols) == 0 || fileTraitName == "" {
			return line, false
		}

		ctx.Verbose("  Moving history imports in %s from generic history to %s_history", relPath(ctx.RootDir, filename), fileTraitName)

		// Build new line with the trait-specific history import
		newLine := prefix + strings.Join(symbols, ", ") + middle + fileTraitName + "_history" + pbSuffix + extension
		return newLine, true
	}

	// Split symbols into history and non-history
	normalSymbols, historyOnlySymbols := splitSymbols(symbols)

	// If no history symbols found, no change needed
	if len(historyOnlySymbols) == 0 {
		return line, false
	}

	ctx.Verbose("  Splitting history imports in %s for trait %s", relPath(ctx.RootDir, filename), traitName)

	// Build new lines
	var newLines []string

	// Original import with non-history symbols (if any)
	if len(normalSymbols) > 0 {
		normalLine := prefix + strings.Join(normalSymbols, ", ") + middle + traitName + pbSuffix + extension
		newLines = append(newLines, normalLine)
	}

	// New import for history symbols
	historyLine := prefix + strings.Join(historyOnlySymbols, ", ") + middle + traitName + "_history" + pbSuffix + extension
	newLines = append(newLines, historyLine)

	return strings.Join(newLines, "\n"), true
}

// processMultiLineImport handles multi-line import statements.
func processMultiLineImport(ctx *fixer.Context, filename string, lines []string, startIdx, endIdx int, fileTraitName string) ([]string, bool) {
	// Extract the end line pattern
	endMatches := multiLineEndPattern.FindStringSubmatch(lines[endIdx])
	if endMatches == nil {
		return lines[startIdx : endIdx+1], false
	}

	indent := endMatches[1]    // Indentation (from the closing brace line)
	middle := endMatches[2]    // "} from '@smart-core-os/sc-bos-ui-gen/proto/"
	traitName := endMatches[3] // "transport" or "electric_history"
	pbSuffix := endMatches[4]  // "_grpc_web_pb" or "_pb"
	extension := endMatches[5] // ".js';" or "';" or "\";"

	// Skip if already a trait-specific history import (ends with _history)
	if strings.HasSuffix(traitName, "_history") {
		return lines[startIdx : endIdx+1], false
	}

	// Collect all symbols from the middle lines, detecting indentation from first symbol line
	var symbols []string
	symbolIndent := ""
	for i := startIdx + 1; i < endIdx; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && symbolIndent == "" {
			// Extract indentation from first non-empty line
			symbolIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		}
		lineSymbols := parseSymbols(trimmed)
		symbols = append(symbols, lineSymbols...)
	}

	// If we didn't detect indentation, use the closing brace indentation
	if symbolIndent == "" {
		symbolIndent = indent
	}

	// Special case: if importing from generic "history" file, all symbols are history-related
	// and we use the trait name determined from the HistoryClient in the file
	if traitName == "history" {
		if len(symbols) == 0 || fileTraitName == "" {
			return lines[startIdx : endIdx+1], false
		}

		ctx.Verbose("  Moving history imports in %s from generic history to %s_history", relPath(ctx.RootDir, filename), fileTraitName)

		// Build new import with the trait-specific history path
		var newLines []string
		newLines = append(newLines, lines[startIdx]) // import {
		for _, sym := range symbols {
			newLines = append(newLines, symbolIndent+sym+",")
		}
		newLines = append(newLines, indent+middle+fileTraitName+"_history"+pbSuffix+extension)

		return newLines, true
	}

	// Split symbols into history and non-history
	normalSymbols, historyOnlySymbols := splitSymbols(symbols)

	// If no history symbols found, no change needed
	if len(historyOnlySymbols) == 0 {
		return lines[startIdx : endIdx+1], false
	}

	ctx.Verbose("  Splitting history imports in %s for trait %s", relPath(ctx.RootDir, filename), traitName)

	var newLines []string

	// Original import with non-history symbols (if any)
	if len(normalSymbols) > 0 {
		newLines = append(newLines, lines[startIdx]) // import {
		for _, sym := range normalSymbols {
			newLines = append(newLines, symbolIndent+sym+",")
		}
		newLines = append(newLines, indent+middle+traitName+pbSuffix+extension)
	}

	// New import for history symbols
	newLines = append(newLines, lines[startIdx]) // import {
	for _, sym := range historyOnlySymbols {
		newLines = append(newLines, symbolIndent+sym+",")
	}
	newLines = append(newLines, indent+middle+traitName+"_history"+pbSuffix+extension)

	return newLines, true
}

// processJSDocImport processes a JSDoc import statement, replacing history imports with trait-specific imports.
// Only replaces imports for history-related types (containing "History" or "Record" in the type name).
// Returns the modified line and true if a change was made.
func processJSDocImport(line, fileTraitName string) (string, bool) {
	// Pattern to extract the type name from JSDoc import
	// Matches: import('...proto/xxx_pb').TypeName or import('...proto/xxx_pb.js').TypeName
	typePattern := regexp.MustCompile(`import\(['"]@smart-core-os/sc-bos-ui-gen/proto/([a-z_]+?)(_(?:grpc_web_)?pb)(?:\.js)?['"]\)\.([A-Za-z]\w+)`)

	// First check if this line contains any history-related types
	hasHistoryType := false
	typeMatches := typePattern.FindAllStringSubmatch(line, -1)
	for _, match := range typeMatches {
		typeName := match[3] // e.g., "ElectricDemandRecord", "ListElectricDemandHistoryRequest", "HealthCheck"
		// Check if this is a history-related type
		if strings.Contains(typeName, "History") || strings.Contains(typeName, "Record") {
			hasHistoryType = true
			break
		}
	}

	// If no history-related types found, don't modify this line
	if !hasHistoryType {
		return line, false
	}

	modified := false
	newLine := line

	matches := jsdocProtoPathPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		protoFile := match[1] // e.g., "history" or "health"
		pbSuffix := match[2]  // e.g., "_pb" or "_grpc_web_pb"

		// For generic "history" imports, use the file's trait name
		if protoFile == "history" {
			oldImport := "proto/history" + pbSuffix
			newImport := "proto/" + fileTraitName + "_history" + pbSuffix
			newLine = strings.ReplaceAll(newLine, oldImport, newImport)
			if newLine != line {
				modified = true
			}
		} else {
			// For trait-specific imports (e.g., health_pb), check if it needs _history suffix
			// Only add _history if it doesn't already have it
			if !strings.Contains(protoFile, "_history") {
				oldImport := "proto/" + protoFile + pbSuffix
				newImport := "proto/" + protoFile + "_history" + pbSuffix
				// Only replace if the import actually appears in the line
				if strings.Contains(line, oldImport) {
					newLine = strings.ReplaceAll(newLine, oldImport, newImport)
					if newLine != line {
						modified = true
					}
				}
			}
		}
	}

	return newLine, modified
}

// detectJSDocImports scans for JSDoc comments with inline import() statements
// from history files that couldn't be automatically replaced.
func detectJSDocImports(ctx *fixer.Context, filename string, lines []string) {
	foundJSDoc := false

	for lineNum, line := range lines {
		// Only check for imports from generic history files (history_pb, history_grpc_web_pb)
		// These are the ones that need the trait name to be replaced
		if strings.Contains(line, "proto/history_pb") || strings.Contains(line, "proto/history_grpc_web_pb") {
			if jsdocImportPattern.MatchString(line) {
				if !foundJSDoc {
					ctx.Info("! Manual JSDoc fix needed in %s: no {trait}HistoryClient import", relPath(ctx.RootDir, filename))
					foundJSDoc = true
				}
				ctx.Info("    Line %d: %s", lineNum+1, strings.TrimSpace(line))
			}
		}
	}
}
