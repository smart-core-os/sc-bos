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

// symbolRelocationEntry describes a symbol that moved to a different package than its parent gentrait package.
type symbolRelocationEntry struct {
	oldGentraitPath string // e.g. "github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	symbol          string // e.g. "Map"
	newImportPath   string // e.g. "github.com/smart-core-os/sc-bos/pkg/util/statusmap"
}

// symbolRelocations lists symbols that moved to a different package than their parent gentrait package
// during the gentrait→proto migration. These are handled as a second pass after the main import remapping.
var symbolRelocations = []symbolRelocationEntry{
	// Map and NewMap moved from pkg/gentrait/statuspb to pkg/util/statusmap to break an import cycle.
	{
		oldGentraitPath: "github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb",
		symbol:          "Map",
		newImportPath:   "github.com/smart-core-os/sc-bos/pkg/util/statusmap",
	},
	{
		oldGentraitPath: "github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb",
		symbol:          "NewMap",
		newImportPath:   "github.com/smart-core-os/sc-bos/pkg/util/statusmap",
	},
}

// transformReferences updates imports and references from pkg/gentrait to pkg/proto.
// This will not work on files that are being moved to the new pkg/proto location.
//
// It updates:
// - imports from pkg/gentrait/* to pkg/proto/*
// - qualified references to use the new package names
func transformReferences(content string) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parsing file: %w", err)
	}

	type change struct {
		oldPath string // should never be empty
		oldName string // empty means no alias
		newPath string // empty means remove import
		newName string // empty means no alias
	}
	var changes []change

	// Track existing pkg/proto imports to detect duplicates,
	// mostly for alias reuse and normalisation.
	existingProtoImports := make(map[string]string) // import path -> package name

	// First pass: collect existing proto imports
	for _, impSpec := range node.Imports {
		if impSpec.Path == nil {
			continue
		}

		importPath := strings.Trim(impSpec.Path.Value, `"`)
		if isProtoImport(importPath) {
			existingProtoImports[importPath] = getImportPackageName(impSpec)
		}
	}

	// processedGentraitPaths tracks which gentrait imports were processed and their alias in this file.
	// Used later to apply symbol-level relocations.
	processedGentraitPaths := make(map[string]string) // gentrait import path -> alias used in file

	// Second pass: analyze imports and plan changes
	for _, impSpec := range node.Imports {
		if impSpec.Path == nil {
			continue
		}

		importPath := strings.Trim(impSpec.Path.Value, `"`)
		if !isGentraitImport(importPath) {
			continue
		}

		oldPath := importPath
		newPath := convertGentraitImportToProto(importPath)
		oldName := getImportPackageName(impSpec)
		newName := gopkg.ImportPathToAssumedName(newPath)

		processedGentraitPaths[oldPath] = oldName

		// Check if the new proto import already exists
		if existingName, exists := existingProtoImports[newPath]; exists {
			// The proto import already exists
			// Special case: if the existing proto import has a gen_{trait}pb alias, remove the alias
			if strings.HasPrefix(existingName, "gen_") && strings.HasSuffix(existingName, "pb") {
				// Remove the gen_ aliased proto import and replace with unaliased version
				changes = append(changes, change{
					oldPath: newPath,
					oldName: existingName,
					newPath: newPath,
					newName: "", // no alias
				})
				// Remove the gentrait import and update its references to canonical name
				changes = append(changes, change{
					oldPath: oldPath,
					oldName: oldName,
					newName: newName,
				})
			} else {
				// Just remove gentrait and update references to use existing proto import
				changes = append(changes, change{
					oldPath: oldPath,
					oldName: oldName,
					newName: existingName, // forces refs to update: oldName.Foo -> existingName.Foo
				})
			}
		} else {
			// Normal case: remove gentrait import and add new proto import
			changes = append(changes, change{
				oldPath: oldPath,
				oldName: oldName,
				newPath: newPath,
				newName: "", // no alias
			})
		}
	}

	if len(changes) == 0 {
		return content, nil
	}

	// Apply changes
	for _, ch := range changes {
		// Remove old import
		canonicalOldName := gopkg.ImportPathToAssumedName(ch.oldPath)
		if ch.oldName != canonicalOldName {
			astutil.DeleteNamedImport(fset, node, ch.oldName, ch.oldPath)
		} else {
			astutil.DeleteImport(fset, node, ch.oldPath)
		}

		// Add new import if specified
		if ch.newPath != "" {
			if ch.newName != "" {
				astutil.AddNamedImport(fset, node, ch.newName, ch.newPath)
			} else {
				astutil.AddImport(fset, node, ch.newPath)
			}
		}

		// Update code references if package name changed
		newName := ch.newName
		if newName == "" {
			newName = gopkg.ImportPathToAssumedName(ch.newPath)
		}
		if ch.oldName != newName {
			astutil.Apply(node, nil, func(c *astutil.Cursor) bool {
				if sel, ok := c.Node().(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						if ident.Name == ch.oldName {
							ident.Name = newName
						}
					}
				}
				return true
			})
		}
	}

	// Apply symbol-level relocations: some symbols moved to different packages than their containing
	// gentrait package. For each processed gentrait import, check for relocated symbols and rewrite them.
	for _, reloc := range symbolRelocations {
		alias, processed := processedGentraitPaths[reloc.oldGentraitPath]
		if !processed {
			continue
		}
		newPkgName := gopkg.ImportPathToAssumedName(reloc.newImportPath)
		rewritten := 0
		astutil.Apply(node, nil, func(c *astutil.Cursor) bool {
			sel, ok := c.Node().(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != alias || sel.Sel.Name != reloc.symbol {
				return true
			}
			ident.Name = newPkgName
			rewritten++
			return true
		})
		if rewritten > 0 {
			astutil.AddImport(fset, node, reloc.newImportPath)
		}
	}

	// Format the updated AST
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return "", fmt.Errorf("printing AST: %w", err)
	}

	// Run goimports to clean up and format
	formatted, err := imports.Process("", buf.Bytes(), &imports.Options{
		FormatOnly: true,
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
	})
	if err != nil {
		return "", fmt.Errorf("formatting: %w", err)
	}

	return string(formatted), nil
}
