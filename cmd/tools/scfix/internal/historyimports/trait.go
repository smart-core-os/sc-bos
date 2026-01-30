package historyimports

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

var (
	// ErrMultipleHistoryClients indicates multiple different HistoryClients were found in the file
	ErrMultipleHistoryClients = errors.New("multiple {trait}HistoryClient imports")
	// ErrNoHistoryClient indicates no HistoryClient was found to determine the trait
	ErrNoHistoryClient = errors.New("no {trait}HistoryClient import")
	// ErrNoHistoryImports indicates no history imports were found in the file
	ErrNoHistoryImports = errors.New("no history imports")
)

// findHistoryClientTrait finds the single HistoryClient symbol in the file and extracts its trait name.
// Returns an error if there are 0 or more than 1 different HistoryClient symbols.
func findHistoryClientTrait(lines []string) (string, error) {
	uniqueTraits := make(map[string]bool)

	// Collect all unique client traits from the file
	for i := 0; i < len(lines); i++ {
		// Try single-line import
		if traits := processSingleLineHistoryImport(lines[i]); len(traits) > 0 {
			for _, trait := range traits {
				uniqueTraits[trait] = true
			}
			continue
		}

		// Try multi-line import
		if traits := processMultiLineHistoryImport(lines, i); len(traits) > 0 {
			for _, trait := range traits {
				uniqueTraits[trait] = true
			}
		}
	}

	// No HistoryClients found
	if len(uniqueTraits) == 0 {
		if fileHasGenericHistoryImports(lines) {
			return "", ErrNoHistoryClient
		}
		return "", ErrNoHistoryImports
	}

	// Multiple different HistoryClients found
	if len(uniqueTraits) > 1 {
		traits := slices.Collect(maps.Keys(uniqueTraits))
		return "", fmt.Errorf("%w: %v", ErrMultipleHistoryClients, traits)
	}

	// Return the single trait name (get the only key from the map)
	for trait := range uniqueTraits {
		return trait, nil
	}

	return "", fmt.Errorf("unexpected error finding trait")
}

// processSingleLineHistoryImport checks if a line is a generic history import and collects client traits
func processSingleLineHistoryImport(line string) []string {
	matches := singleLineImportPattern.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	traitName := matches[4]
	if traitName != "history" {
		return nil
	}

	symbolsStr := matches[2]
	symbols := parseSymbols(symbolsStr)
	return collectClientTraitsFromSymbols(symbols)
}

// processMultiLineHistoryImport checks if a line starts a generic history import and collects client traits
func processMultiLineHistoryImport(lines []string, startIdx int) []string {
	if !multiLineStartPattern.MatchString(lines[startIdx]) {
		return nil
	}

	endIdx := findMultiLineImportEnd(lines, startIdx)
	if endIdx <= startIdx {
		return nil
	}

	endMatches := multiLineEndPattern.FindStringSubmatch(lines[endIdx])
	if endMatches == nil {
		return nil
	}

	traitName := endMatches[3]
	if traitName != "history" {
		return nil
	}

	// Collect symbols from middle lines
	var allSymbols []string
	for j := startIdx + 1; j < endIdx; j++ {
		trimmed := strings.TrimSpace(lines[j])
		symbols := parseSymbols(trimmed)
		allSymbols = append(allSymbols, symbols...)
	}

	return collectClientTraitsFromSymbols(allSymbols)
}

// collectClientTraitsFromSymbols extracts trait names from HistoryClient symbols
func collectClientTraitsFromSymbols(symbols []string) []string {
	var traits []string
	for _, sym := range symbols {
		if !strings.HasSuffix(sym, "Client") && !strings.HasSuffix(sym, "PromiseClient") {
			continue
		}
		trait := extractTraitFromHistorySymbol(sym)
		if trait != "" {
			traits = append(traits, trait)
		}
	}
	return traits
}

// fileHasGenericHistoryImports checks if the file has any imports from generic history files
func fileHasGenericHistoryImports(lines []string) bool {
	for i := 0; i < len(lines); i++ {
		if hasGenericHistoryImport(lines[i]) {
			return true
		}
		if hasGenericHistoryImportMultiLine(lines, i) {
			return true
		}
	}
	return false
}

// hasGenericHistoryImport checks if a line has any import from generic history files
func hasGenericHistoryImport(line string) bool {
	if matches := singleLineImportPattern.FindStringSubmatch(line); matches != nil {
		return matches[4] == "history"
	}
	return false
}

// hasGenericHistoryImportMultiLine checks if a multi-line import is from generic history files
func hasGenericHistoryImportMultiLine(lines []string, startIdx int) bool {
	if !multiLineStartPattern.MatchString(lines[startIdx]) {
		return false
	}

	endIdx := findMultiLineImportEnd(lines, startIdx)
	if endIdx <= startIdx {
		return false
	}

	endMatches := multiLineEndPattern.FindStringSubmatch(lines[endIdx])
	if endMatches == nil {
		return false
	}

	return endMatches[3] == "history"
}
