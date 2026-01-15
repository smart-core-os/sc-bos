package goprotoimports

import "strings"

// inferTraitPackage returns the new import path, package name, and symbol for a given gen type name.
// The typeName argument should be the unqualified name of a symbol from pkg/gen (e.g., "WrapApi").
// It returns ok=false if the type is not found in the mapping.
func inferTraitPackage(typeName string) (importPath, pkgName, symbol string, ok bool) {
	// Look up using lowercase to make it case-insensitive
	fullEntry, found := typeToTraitMap[strings.ToLower(typeName)]
	if !found {
		return "", "", "", false
	}

	// fullEntry is in format "importPath.symbol" (e.g., "meterpb.WrapApi" or "driver/dalipb.AddToGroupRequest")
	// We need to split this into importPath and symbol, then extract the package name from the import path

	// Find the last dot to separate the symbol from the path
	lastDot := strings.LastIndex(fullEntry, ".")
	if lastDot == -1 {
		// Shouldn't happen with properly generated map, but handle gracefully
		return fullEntry, fullEntry, typeName, true
	}

	importPath = fullEntry[:lastDot]
	symbol = fullEntry[lastDot+1:]

	// Extract package name from import path (last segment after /)
	lastSlash := strings.LastIndex(importPath, "/")
	if lastSlash == -1 {
		// No slash, so import path is the package name
		pkgName = importPath
	} else {
		// Extract last segment after the last slash
		pkgName = importPath[lastSlash+1:]
	}

	return importPath, pkgName, symbol, true
}
