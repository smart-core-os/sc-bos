package historyimports

import "strings"

// isHistorySymbol returns true if the symbol name indicates it's related to history functionality.
func isHistorySymbol(symbol string) bool {
	return strings.Contains(symbol, "History")
}

// extractTraitFromHistorySymbol extracts the trait name from a history symbol.
// E.g., "AirQualitySensorHistory" -> "air_quality_sensor"
//
//	"ListAirQualityHistoryRequest" -> "air_quality"
//	"TransportHistoryPromiseClient" -> "transport"
func extractTraitFromHistorySymbol(symbol string) string {
	// Remove common suffixes
	symbol = strings.TrimSuffix(symbol, "PromiseClient")
	symbol = strings.TrimSuffix(symbol, "Client")
	symbol = strings.TrimSuffix(symbol, "Request")
	symbol = strings.TrimSuffix(symbol, "Response")

	// Find "History" and extract everything before it
	if idx := strings.Index(symbol, "History"); idx > 0 {
		traitPart := symbol[:idx]
		// Remove common prefixes like "List", "Get", "Pull", etc.
		traitPart = strings.TrimPrefix(traitPart, "List")
		traitPart = strings.TrimPrefix(traitPart, "Get")
		traitPart = strings.TrimPrefix(traitPart, "Pull")
		traitPart = strings.TrimPrefix(traitPart, "Create")
		traitPart = strings.TrimPrefix(traitPart, "Update")
		traitPart = strings.TrimPrefix(traitPart, "Delete")

		// Convert from PascalCase to snake_case
		return toSnakeCase(traitPart)
	}

	return ""
}

// toSnakeCase converts PascalCase to snake_case.
// E.g., "AirQualitySensor" -> "air_quality_sensor"
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// parseSymbols parses comma-separated import symbols, handling whitespace
func parseSymbols(symbolsStr string) []string {
	parts := strings.Split(symbolsStr, ",")
	var symbols []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			symbols = append(symbols, trimmed)
		}
	}
	return symbols
}

// splitSymbols splits symbols into normal and history-related symbols.
func splitSymbols(symbols []string) (normal, history []string) {
	for _, sym := range symbols {
		if isHistorySymbol(sym) {
			history = append(history, sym)
		} else {
			normal = append(normal, sym)
		}
	}
	return normal, history
}
