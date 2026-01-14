package goprotoimports

import (
	"go/ast"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/go/gopkg"
)

// findUsedGenTypes finds all types from pkg/gen that are referenced in the file.
func findUsedGenTypes(node *ast.File) map[string]bool {
	types := make(map[string]bool)

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for selector expressions like gen.SomeType
		selExpr, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Check if the selector is on "gen" package
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok || ident.Name != "gen" {
			return true
		}

		// Record the type name
		types[selExpr.Sel.Name] = true

		return true
	})

	return types
}

// findExistingTraitImports returns a mapping from package identifier to import path.
// For example, {"meterpb": "github.com/smart-core-os/sc-bos/pkg/proto/meterpb"}.
func findExistingTraitImports(node *ast.File) map[string]string {
	existing := make(map[string]string) // package identifier -> import path

	for _, imp := range node.Imports {
		// Get the package identifier
		var pkgName string
		if imp.Name != nil {
			// Explicit alias
			pkgName = imp.Name.Name
		} else {
			// Derive from import path (last segment)
			p := strings.Trim(imp.Path.Value, `"`)
			pkgName = gopkg.ImportPathToAssumedName(p)
		}

		if pkgName != "" && pkgName != "C" && pkgName != "_" && pkgName != "." {
			// Store the full import path (without quotes)
			importPath := imp.Path.Value
			if len(importPath) > 2 {
				importPath = importPath[1 : len(importPath)-1] // Remove quotes
			}
			existing[pkgName] = importPath
		}
	}

	return existing
}
