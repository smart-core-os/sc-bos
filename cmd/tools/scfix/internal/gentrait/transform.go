package gentrait

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/go/gopkg"
)

const (
	gentraitImportPrefix = "github.com/smart-core-os/sc-bos/pkg/gentrait/"
	protoImportPrefix    = "github.com/smart-core-os/sc-bos/pkg/proto/"
)

// transformFileContent updates package declaration and imports in the file content.
// Argument relPath is the file path relative to the gentrait directory.
//
// It updates:
// - package declarations to match the new package name (e.g., "package meter" -> "package meterpb")
// - imports from pkg/gentrait/* to pkg/proto/*
// - removes self-imports (imports to the same package the file is being moved to)
// - updates qualified references to removed self-imports to be unqualified
func transformFileContent(content, relPath string) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parsing file: %w", err)
	}

	newPkgName, newImportPath := relPathToProtoImport(relPath)

	type importChange struct {
		add  bool // true for add, false for remove
		path string
		name string // empty for imports without alias
	}

	type codeChange struct {
		oldQualifier string // package qualifier to replace or remove
		newQualifier string // new qualifier, or empty to remove entirely
	}

	var importChanges []importChange
	var codeChanges []codeChange
	seenImports := make(map[string]bool) // track imports we're adding to deduplicate

	// Update package name and doc comment if needed
	var pkgNameChanged bool
	if newPkgName != "" && node.Name.Name != newPkgName {
		oldPkgName := node.Name.Name
		node.Name.Name = newPkgName
		pkgNameChanged = true

		// Update package doc comment to match new package name
		if node.Doc != nil {
			for _, comment := range node.Doc.List {
				if strings.HasPrefix(comment.Text, "// Package ") {
					comment.Text = strings.Replace(comment.Text, "// Package "+oldPkgName, "// Package "+newPkgName, 1)
				} else if strings.HasPrefix(comment.Text, "/* Package ") {
					comment.Text = strings.Replace(comment.Text, "/* Package "+oldPkgName, "/* Package "+newPkgName, 1)
				}
			}
		}
	}

	for _, impSpec := range node.Imports {
		if impSpec.Path == nil {
			continue
		}

		importPath := strings.Trim(impSpec.Path.Value, `"`)
		pkgName := getImportPackageName(impSpec)

		switch {
		case isGentraitImport(importPath):
			newImport := convertGentraitImportToProto(importPath)
			importChanges = append(importChanges, importChange{add: false, path: importPath})

			if newImport == newImportPath {
				// This becomes a self-import - remove the qualifier from code
				codeChanges = append(codeChanges, codeChange{oldQualifier: pkgName, newQualifier: ""})
			} else {
				// Add the new proto import
				importChanges = append(importChanges, importChange{add: true, path: newImport})
				seenImports[newImport] = true
			}

		case isProtoImport(importPath):
			switch {
			case importPath == newImportPath:
				// Already a self-import - remove it and the qualifier
				var aliasName string
				if impSpec.Name != nil {
					aliasName = impSpec.Name.Name
				}
				importChanges = append(importChanges, importChange{add: false, path: importPath, name: aliasName})
				codeChanges = append(codeChanges, codeChange{oldQualifier: pkgName, newQualifier: ""})

			case impSpec.Name != nil && seenImports[importPath]:
				// Aliased proto import that duplicates one we're adding
				canonicalName := gopkg.ImportPathToAssumedName(importPath)

				if pkgName != canonicalName {
					// Remove the aliased import and update code to use canonical name
					importChanges = append(importChanges, importChange{add: false, path: importPath, name: pkgName})
					codeChanges = append(codeChanges, codeChange{oldQualifier: pkgName, newQualifier: canonicalName})
				}
			}
		}
	}

	// Apply import changes
	for _, change := range importChanges {
		if change.add {
			astutil.AddImport(fset, node, change.path)
		} else {
			if change.name != "" {
				// Aliased import - use DeleteNamedImport
				astutil.DeleteNamedImport(fset, node, change.name, change.path)
			} else {
				// Regular import
				astutil.DeleteImport(fset, node, change.path)
			}
		}
	}

	// Apply code changes
	if len(codeChanges) > 0 {
		astutil.Apply(node, nil, func(c *astutil.Cursor) bool {
			if sel, ok := c.Node().(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					for _, change := range codeChanges {
						if ident.Name == change.oldQualifier {
							if change.newQualifier == "" {
								// Remove qualifier entirely
								c.Replace(sel.Sel)
							} else {
								// Replace with new qualifier
								ident.Name = change.newQualifier
							}
							break
						}
					}
				}
			}
			return true
		})
	}

	if !pkgNameChanged && len(importChanges) == 0 && len(codeChanges) == 0 {
		return content, nil
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return "", fmt.Errorf("printing AST: %w", err)
	}

	formatted, err := imports.Process("", buf.Bytes(), &imports.Options{
		FormatOnly: true, // stops new imports being added, we've added them all
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
	})
	if err != nil {
		return "", fmt.Errorf("formatting: %w", err)
	}

	return string(formatted), nil
}

// getImportPackageName returns the package name for an import spec.
// If the import has an alias, returns the alias. Otherwise derives the assumed name from the path.
func getImportPackageName(impSpec *ast.ImportSpec) string {
	if impSpec.Name != nil {
		return impSpec.Name.Name
	}
	importPath := strings.Trim(impSpec.Path.Value, `"`)
	return gopkg.ImportPathToAssumedName(importPath)
}

// isGentraitImport checks if the import path is from pkg/gentrait.
func isGentraitImport(importPath string) bool {
	return strings.HasPrefix(importPath, gentraitImportPrefix)
}

// isProtoImport checks if the import path is from pkg/proto.
func isProtoImport(importPath string) bool {
	return strings.HasPrefix(importPath, protoImportPrefix)
}

// convertGentraitImportToProto converts a gentrait import path to its proto equivalent.
// Returns the new proto import path.
func convertGentraitImportToProto(importPath string) string {
	gentraitPath := strings.TrimPrefix(importPath, gentraitImportPrefix)
	protoPath := gentraitToProtoImportPath(gentraitPath)
	return protoImportPrefix + protoPath
}
