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
	"regexp"
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

// skipDirs lists directory path components that goprotoimports should not modify.
// sc-api/ contains generated or upstream-managed files that are not part of the sc-bos codebase.
var skipDirs = []string{"/sc-api/"}

func run(ctx *fixer.Context) (int, error) {
	totalChanges := 0

	err := filepath.WalkDir(ctx.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for _, skip := range skipDirs {
			if strings.Contains(path, skip) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
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

	bosImportPath := func(paths ...string) string {
		return "github.com/smart-core-os/sc-bos/" + path.Join(paths...)
	}

	// All source packages map to pkg/proto/*pb destinations.
	// defaultAlias is the Go package name used when no explicit import alias is given.
	// directDest, if non-empty, bypasses the typemap: all symbols map directly to that
	// destination package (e.g. "typespb"), no per-symbol lookup is needed.
	sourcePackages := []struct {
		defaultAlias string
		importPath   string
		directDest   string // non-empty: bypass typemap, map directly to this pkg name
	}{
		{"gen", bosImportPath("pkg/gen"), ""},
		{"traits", bosImportPath("sc-api/go/traits"), ""},
		{"types", bosImportPath("sc-api/go/types"), "typespb"},
		{"time", bosImportPath("sc-api/go/types/time"), "timepb"},
		{"info", bosImportPath("sc-api/go/info"), "infopb"},
	}

	// importAliases maps import path → the alias actually used in this file.
	// This handles files that explicitly alias an import (e.g. timepb "sc-api/go/types/time").
	importAliases := make(map[string]string)
	for _, imp := range node.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		for _, src := range sourcePackages {
			if p == src.importPath {
				if imp.Name != nil {
					importAliases[p] = imp.Name.Name
				} else {
					importAliases[p] = src.defaultAlias
				}
			}
		}
	}

	if len(importAliases) == 0 {
		return 0, nil
	}

	totalChanges := 0

	for _, src := range sourcePackages {
		alias, present := importAliases[src.importPath]
		if !present {
			continue
		}
		var c int
		if src.directDest != "" {
			c, err = migrateDirectPackageRefs(ctx, fset, node, filename, alias, src.importPath,
				src.directDest, bosImportPath)
		} else {
			c, err = migratePackageRefs(ctx, fset, node, filename, alias, src.importPath,
				bosImportPath)
		}
		if err != nil {
			return totalChanges, err
		}
		totalChanges += c
	}

	if totalChanges == 0 {
		return 0, nil
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

	ctx.Verbose("  Modified %s (%d changes)", filepath.Base(filename), totalChanges)
	return totalChanges, nil
}

// migratePackageRefs rewrites selector expressions of the form `pkgAlias.TypeName`
// to the new per-trait packages via inferTraitPackage.
// oldImportPath is removed; new per-trait imports are added.
// Types whose destination package equals the file's own package are rewritten to
// bare identifiers (no qualifier) and no self-import is added.
func migratePackageRefs(
	ctx *fixer.Context,
	fset *token.FileSet,
	node *ast.File,
	filename string,
	pkgAlias string,
	oldImportPath string,
	bosImportPath func(...string) string,
) (int, error) {
	// Find all types used via this package alias.
	usedTypes := findUsedTypesByAlias(node, pkgAlias)
	if len(usedTypes) == 0 {
		return 0, nil
	}

	typeToPackageName := make(map[string]string)
	typeToImportPath := make(map[string]string)
	typeToNewSymbol := make(map[string]string)
	var unknownTypes []string
	for typeName := range usedTypes {
		ip, pkg, sym, ok := inferTraitPackage(typeName)
		if ok {
			typeToPackageName[typeName] = pkg
			typeToImportPath[typeName] = ip
			typeToNewSymbol[typeName] = sym
		} else {
			unknownTypes = append(unknownTypes, typeName)
		}
	}

	if len(unknownTypes) > 0 {
		ctx.Info("! Warning: Cannot determine trait package for types in %s: %v", filepath.Base(filename), unknownTypes)
	}

	if len(typeToPackageName) == 0 {
		return 0, nil
	}

	// Determine the file's own import path to detect self-imports.
	ownImportPath, err := fileImportPath(ctx.RootDir, filename)
	if err != nil {
		// If we can't determine the own path, fall back to no self-import detection.
		ownImportPath = ""
	}

	// Classify types: self-types live in the same package as the file being processed.
	isSelfType := make(map[string]bool)
	for typeName, ip := range typeToImportPath {
		if ownImportPath != "" && bosImportPath("pkg/proto", ip) == ownImportPath {
			isSelfType[typeName] = true
		}
	}

	neededPackages := make(map[string]string) // pkg name -> import path
	for typeName := range usedTypes {
		if isSelfType[typeName] {
			continue // self-types need no import
		}
		if pkgName, ok := typeToPackageName[typeName]; ok {
			neededPackages[pkgName] = typeToImportPath[typeName]
		}
	}

	existingImports := findExistingTraitImports(node)
	packageToAlias := make(map[string]string)
	for pkgName, ip := range neededPackages {
		expectedFullPath := bosImportPath("pkg/proto", ip)
		if existingPath, exists := existingImports[pkgName]; exists {
			if existingPath != expectedFullPath {
				packageToAlias[pkgName] = pkgName + "2"
			}
		}
	}

	modified := false
	changes := 0

	astutil.Apply(node, func(cursor *astutil.Cursor) bool {
		selExpr, ok := cursor.Node().(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok || ident.Name != pkgAlias {
			return true
		}

		typeName := selExpr.Sel.Name
		if isSelfType[typeName] {
			// Replace pkgAlias.TypeName with just TypeName (bare identifier).
			newSymbol := typeToNewSymbol[typeName]
			selExpr.Sel.Name = newSymbol
			cursor.Replace(selExpr.Sel)
			modified = true
			changes++
		} else if pkgName, ok := typeToPackageName[typeName]; ok {
			newSymbol := typeToNewSymbol[typeName]
			if alias, hasAlias := packageToAlias[pkgName]; hasAlias {
				ident.Name = alias
			} else {
				ident.Name = pkgName
			}
			selExpr.Sel.Name = newSymbol
			modified = true
			changes++
		}
		return true
	}, nil)

	if !modified {
		return 0, nil
	}

	// Update comments that reference pkgAlias.Type
	for _, commentGroup := range node.Comments {
		for _, comment := range commentGroup.List {
			updated := updateCommentGenReferences(comment.Text, typeToPackageName, typeToNewSymbol, packageToAlias)
			if updated != comment.Text {
				comment.Text = updated
				changes++
			}
		}
	}

	// Update imports: remove old, add new per-trait imports.
	deleteSourceImport(fset, node, oldImportPath)
	for pkgName, ip := range neededPackages {
		fullImportPath := bosImportPath("pkg/proto", ip)
		if alias, hasAlias := packageToAlias[pkgName]; hasAlias {
			astutil.AddNamedImport(fset, node, alias, fullImportPath)
		} else {
			astutil.AddImport(fset, node, fullImportPath)
		}
	}

	return changes, nil
}

// migrateDirectPackageRefs rewrites selector expressions of the form `pkgAlias.Symbol`
// to `newPkgName.Symbol` without consulting the typemap — all symbols map directly to
// newPkgName (e.g. "typespb"). The symbol name is preserved unchanged.
// oldImportPath is removed; a single new import for bosImportPath("pkg/proto", newPkgName) is added.
// If the file lives in the destination package itself, qualifiers are dropped entirely.
func migrateDirectPackageRefs(
	ctx *fixer.Context,
	fset *token.FileSet,
	node *ast.File,
	filename string,
	pkgAlias string,
	oldImportPath string,
	newPkgName string,
	bosImportPath func(...string) string,
) (int, error) {
	usedTypes := findUsedTypesByAlias(node, pkgAlias)
	if len(usedTypes) == 0 {
		// No usages of the old alias — the import is a stale artifact from a partial migration.
		// Remove it so the file compiles cleanly.
		deleteSourceImport(fset, node, oldImportPath)
		return 1, nil
	}

	newImportPath := bosImportPath("pkg/proto", newPkgName)

	// Determine the file's own import path to detect self-imports.
	ownImportPath, err := fileImportPath(ctx.RootDir, filename)
	if err != nil {
		ownImportPath = ""
	}
	isSelf := ownImportPath != "" && newImportPath == ownImportPath

	// Check whether the destination package name is already occupied by a different import.
	existingImports := findExistingTraitImports(node)
	destAlias := newPkgName
	if !isSelf {
		if existingPath, exists := existingImports[newPkgName]; exists && existingPath != newImportPath && existingPath != oldImportPath {
			destAlias = newPkgName + "2"
		}
	}

	changes := 0
	astutil.Apply(node, func(cursor *astutil.Cursor) bool {
		selExpr, ok := cursor.Node().(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok || ident.Name != pkgAlias {
			return true
		}
		if isSelf {
			// Drop qualifier entirely — replace pkg.Symbol with just Symbol.
			cursor.Replace(selExpr.Sel)
		} else {
			ident.Name = destAlias
		}
		changes++
		return true
	}, nil)

	if changes == 0 {
		return 0, nil
	}

	// Update comments that reference pkgAlias.Symbol.
	commentPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(pkgAlias) + `\.([A-Z][a-zA-Z0-9_]*)`)
	for _, commentGroup := range node.Comments {
		for _, comment := range commentGroup.List {
			updated := commentPattern.ReplaceAllStringFunc(comment.Text, func(match string) string {
				symbol := commentPattern.FindStringSubmatch(match)[1]
				if isSelf {
					return symbol
				}
				return destAlias + "." + symbol
			})
			if updated != comment.Text {
				comment.Text = updated
				changes++
			}
		}
	}

	// Update imports: remove old, add new (unless self-import).
	deleteSourceImport(fset, node, oldImportPath)
	if !isSelf {
		if destAlias != newPkgName {
			astutil.AddNamedImport(fset, node, destAlias, newImportPath)
		} else {
			astutil.AddImport(fset, node, newImportPath)
		}
	}

	return changes, nil
}

// findUsedTypesByAlias finds all type names used via a specific package alias
// (e.g., "gen" or "traits") in selector expressions.
func findUsedTypesByAlias(node *ast.File, pkgAlias string) map[string]bool {
	types := make(map[string]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		selExpr, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok || ident.Name != pkgAlias {
			return true
		}
		types[selExpr.Sel.Name] = true
		return true
	})
	return types
}

// deleteSourceImport removes an import by path regardless of whether it has an explicit alias.
// astutil.DeleteImport (via DeleteNamedImport with name="") only removes unaliased imports;
// for aliased imports like timepb "pkg/path", DeleteNamedImport must be called with the alias.
func deleteSourceImport(fset *token.FileSet, node *ast.File, importPath string) {
	for _, imp := range node.Imports {
		if strings.Trim(imp.Path.Value, `"`) == importPath {
			if imp.Name != nil {
				astutil.DeleteNamedImport(fset, node, imp.Name.Name, importPath)
			} else {
				astutil.DeleteImport(fset, node, importPath)
			}
			return
		}
	}
}

// fileImportPath returns the Go import path for the directory containing filename,
// computed by reading the module name from go.mod in rootDir.
func fileImportPath(rootDir, filename string) (string, error) {
	goModBytes, err := os.ReadFile(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	var moduleName string
	for _, line := range strings.Split(string(goModBytes), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			moduleName = strings.TrimPrefix(line, "module ")
			moduleName = strings.TrimSpace(moduleName)
			break
		}
	}
	if moduleName == "" {
		return "", fmt.Errorf("module directive not found in go.mod")
	}
	fileDir := filepath.Dir(filename)
	relDir, err := filepath.Rel(rootDir, fileDir)
	if err != nil {
		return "", fmt.Errorf("computing relative path: %w", err)
	}
	return moduleName + "/" + filepath.ToSlash(relDir), nil
}
