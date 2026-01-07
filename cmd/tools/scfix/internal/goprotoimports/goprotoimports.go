// Package goprotoimports updates Go imports from pkg/gen to pkg/proto/{trait}pb.
// The protogopkg fixer changes proto files to update the target go package,
// this fixer updates go code that uses those generated types to refer to the new packages and symbols.
package goprotoimports

// This fixer relies on the output of the genprototypemap tool to generate
// a mapping from [gen.]TypeName to the new scoped package and symbol name.
// The generator must be run on a sc-bos repo that has had the protogopkg
// fixer and genproto tools run against it.

//go:generate go run ./genprototypemap

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "goprotoimports",
	Desc: "Update Go imports from pkg/gen to pkg/proto/{trait}pb",
	Run:  run,
}

func run(ctx *fixer.Context) (int, error) {
	totalChanges := 0

	err := filepath.WalkDir(ctx.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if !fixer.ShouldProcessFile(path, info) {
			return nil
		}

		changes, err := processFile(ctx, path)
		if err != nil {
			return fmt.Errorf("processing %s: %w", path, err)
		}
		totalChanges += changes
		return nil
	})

	return totalChanges, err
}

func processFile(ctx *fixer.Context, filename string) (int, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return 0, err
	}

	// Check if file imports pkg/gen from sc-bos
	// Note: The gen package is always from sc-bos, regardless of which repo we're running against
	hasGenImport := false
	bosImportPath := func(paths ...string) string {
		return "github.com/smart-core-os/sc-bos/" + path.Join(paths...)
	}
	genImportPath := bosImportPath("pkg/gen")
	for _, imp := range node.Imports {
		if strings.Trim(imp.Path.Value, `"`) == genImportPath {
			hasGenImport = true
			break
		}
	}

	if !hasGenImport {
		return 0, nil
	}

	// Analyze the file to determine which types from pkg/gen are used
	usedTypes := findUsedGenTypes(node)
	if len(usedTypes) == 0 {
		return 0, nil
	}

	// Determine which trait packages are needed and build type->package mapping
	typeToPackageName := make(map[string]string) // maps old symbol to new package name (for code references)
	typeToImportPath := make(map[string]string)  // maps old symbol to import path
	typeToNewSymbol := make(map[string]string)   // maps old symbol to new symbol name
	var unknownTypes []string
	for typeName := range usedTypes {
		importPath, pkgName, newSymbol, ok := inferTraitPackage(typeName)
		if ok {
			typeToPackageName[typeName] = pkgName
			typeToImportPath[typeName] = importPath
			typeToNewSymbol[typeName] = newSymbol
		} else {
			unknownTypes = append(unknownTypes, typeName)
		}
	}

	// Warn about types we can't transform
	if len(unknownTypes) > 0 {
		ctx.Info("! Warning: Cannot determine trait package for types in %s: %v", filepath.Base(filename), unknownTypes)
	}

	if len(typeToPackageName) == 0 {
		// If we can't determine any packages, leave file unchanged
		return 0, nil
	}

	// Collect unique packages needed (by package name for collision detection)
	neededPackages := make(map[string]string) // package name -> import path
	for typeName := range usedTypes {
		if pkgName, ok := typeToPackageName[typeName]; ok {
			importPath := typeToImportPath[typeName]
			neededPackages[pkgName] = importPath
		}
	}

	// Check for existing imports to detect collisions
	existingImports := findExistingTraitImports(node)

	// Determine if we need to use an alias for gen -> trait package references
	// This typically happens when a file imports both pkg/gen and pkg/genproto.
	// For example meterpb.Model and gen.MeterReading both would use "meterpb" package name.
	packageToAlias := make(map[string]string)
	for pkgName, importPath := range neededPackages {
		expectedFullPath := bosImportPath("pkg/proto", importPath)
		if existingPath, exists := existingImports[pkgName]; exists {
			// Package name is already in use
			if existingPath != expectedFullPath {
				// It's imported from a different path - need an alias to avoid collision
				// Use "gen_{pkgName}" format to avoid conflicts
				packageToAlias[pkgName] = "gen_" + pkgName
			}
			// else: it's the exact same import, no alias needed, we'll skip adding it
		}
	}

	modified := false
	changes := 0

	// Update all gen.Type references to pkgName.NewType (or gen_pkgName.NewType if aliased)
	ast.Inspect(node, func(n ast.Node) bool {
		selExpr, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok || ident.Name != "gen" {
			return true
		}

		// Look up which package and new symbol this type should use
		typeName := selExpr.Sel.Name
		if pkgName, ok := typeToPackageName[typeName]; ok {
			newSymbol := typeToNewSymbol[typeName]

			// Update the package identifier
			if alias, hasAlias := packageToAlias[pkgName]; hasAlias {
				ident.Name = alias
			} else {
				ident.Name = pkgName
			}

			// Update the selector to use the new symbol name
			selExpr.Sel.Name = newSymbol

			modified = true
			changes++
		}

		return true
	})

	if !modified {
		return 0, nil
	}

	// Update comments that reference gen.Type
	for _, commentGroup := range node.Comments {
		for _, comment := range commentGroup.List {
			updated := updateCommentGenReferences(comment.Text, typeToPackageName, typeToNewSymbol, packageToAlias)
			if updated != comment.Text {
				comment.Text = updated
				changes++
			}
		}
	}

	// Update imports: remove gen, add trait packages
	astutil.DeleteImport(fset, node, genImportPath)
	for pkgName, importPath := range neededPackages {
		fullImportPath := bosImportPath("pkg/proto", importPath)
		if alias, hasAlias := packageToAlias[pkgName]; hasAlias {
			astutil.AddNamedImport(fset, node, alias, fullImportPath)
		} else {
			astutil.AddImport(fset, node, fullImportPath)
		}
	}

	if !ctx.DryRun {
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, node); err != nil {
			return 0, fmt.Errorf("formatting AST: %w", err)
		}

		// LocalPrefix is equivalent to goimports -local flag.
		// This setting matches how we format our projects.
		imports.LocalPrefix = "github.com/vanti-dev,github.com/smart-core-os,github.com/kahu-work"
		formatted, err := imports.Process(filename, buf.Bytes(), nil)
		if err != nil {
			return 0, fmt.Errorf("formatting source: %w", err)
		}

		if err := os.WriteFile(filename, formatted, 0644); err != nil {
			return 0, fmt.Errorf("writing file: %w", err)
		}
	}

	ctx.Verbose("  Modified %s (%d changes)", filepath.Base(filename), changes)

	return changes, nil
}
