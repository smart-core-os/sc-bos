package goprotoimports

import (
	"regexp"
	"strings"
)

// genRefPattern matches gen.TypeName patterns in comments
var genRefPattern = regexp.MustCompile(`\bgen\.([A-Z][a-zA-Z0-9_]*)`)

// updateCommentGenReferences updates references to gen.Type in comments.
// Handles both regular comments and doc comments like [gen.Type].
func updateCommentGenReferences(commentText string, typeToPackageName, typeToNewSymbol, packageToAlias map[string]string) string {
	updated := commentText

	// Find all gen.TypeName patterns in the comment using the pre-compiled regex
	updated = genRefPattern.ReplaceAllStringFunc(updated, func(match string) string {
		// Extract the type name (everything after "gen.")
		typeName := strings.TrimPrefix(match, "gen.")

		// First check if this type was used in the code (in our map)
		if pkgName, ok := typeToPackageName[typeName]; ok {
			newSymbol := typeToNewSymbol[typeName]
			if alias, hasAlias := packageToAlias[pkgName]; hasAlias {
				return alias + "." + newSymbol
			}
			return pkgName + "." + newSymbol
		}

		// If not in our map, try looking it up in the global typemap
		if _, pkgName, symbol, ok := inferTraitPackage(typeName); ok {
			// Check if we have an alias for this package
			if alias, hasAlias := packageToAlias[pkgName]; hasAlias {
				return alias + "." + symbol
			}
			return pkgName + "." + symbol
		}

		// Can't find it, leave as-is
		return match
	})

	return updated
}
