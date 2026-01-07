package goprotoimports

import (
	"go/ast"
	"path"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
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
			pkgName = importPathToAssumedName(p)
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

// importPathToAssumedName returns the assumed package name of an import path.
// This is copied from x/tools /internal/imports/fix.go.
func importPathToAssumedName(importPath string) string {
	base := path.Base(importPath)
	if strings.HasPrefix(base, "v") {
		if _, err := strconv.Atoi(base[1:]); err == nil {
			dir := path.Dir(importPath)
			if dir != "." {
				base = path.Base(dir)
			}
		}
	}
	base = strings.TrimPrefix(base, "go-")
	if i := strings.IndexFunc(base, notIdentifier); i >= 0 {
		base = base[:i]
	}
	return base
}

// notIdentifier reports whether ch is an invalid identifier character.
func notIdentifier(ch rune) bool {
	return !('a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' ||
		'0' <= ch && ch <= '9' ||
		ch == '_' ||
		ch >= utf8.RuneSelf && (unicode.IsLetter(ch) || unicode.IsDigit(ch)))
}
