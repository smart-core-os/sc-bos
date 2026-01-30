// Package wrap converts pkg.WrapFoo calls to New*Client with wrap.ServerToClient.
package wrap

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "wrap",
	Desc: "Convert pkg.WrapFoo calls to New*Client with wrap.ServerToClient",
	Run:  run,
}

func run(ctx *fixer.Context) (int, error) {
	totalChanges := 0

	err := filepath.Walk(ctx.RootDir, func(path string, info os.FileInfo, err error) error {
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

	var changes int
	modified := false
	needsWrapImport := false
	needsTraitsImport := false

	packageInfo := make(map[string]*packageWrapInfo)
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			parts := strings.Split(path, "/")
			pkgName = parts[len(parts)-1]
		}

		if strings.Contains(path, "/sc-golang/pkg/trait/") ||
			strings.Contains(path, "/sc-bos/pkg/gentrait/") ||
			path == "github.com/smart-core-os/sc-api/go/traits" ||
			path == "github.com/smart-core-os/sc-bos/pkg/gen" {
			packageInfo[pkgName] = &packageWrapInfo{
				importPath: path,
				pkgName:    pkgName,
			}
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		pkgIdent, ok := selExpr.X.(*ast.Ident)
		if !ok {
			return true
		}

		pkgInfo, exists := packageInfo[pkgIdent.Name]
		if !exists {
			return true
		}

		funcName := selExpr.Sel.Name
		wrapInfo := getWrapInfo(pkgInfo, funcName)
		if wrapInfo == nil {
			return true
		}

		ctx.Verbose("  Found %s.%s in %s", pkgIdent.Name, funcName, filepath.Base(filename))

		if len(callExpr.Args) > 0 {
			callExpr.Fun = &ast.SelectorExpr{
				X:   ast.NewIdent(wrapInfo.targetPkg),
				Sel: ast.NewIdent(wrapInfo.clientConstructor),
			}

			serverArg := callExpr.Args[0]
			callExpr.Args = []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("wrap"),
						Sel: ast.NewIdent("ServerToClient"),
					},
					Args: []ast.Expr{
						&ast.SelectorExpr{
							X:   ast.NewIdent(wrapInfo.targetPkg),
							Sel: ast.NewIdent(wrapInfo.serviceDesc),
						},
						serverArg,
					},
				},
			}

			modified = true
			changes++
			needsWrapImport = true
			if wrapInfo.needsTraitsImport {
				needsTraitsImport = true
			}
		}

		return true
	})

	if !modified {
		return 0, nil
	}

	if !ctx.DryRun {
		if needsWrapImport {
			ensureImport(node, "github.com/smart-core-os/sc-golang/pkg/wrap")
		}
		if needsTraitsImport {
			ensureImport(node, "github.com/smart-core-os/sc-api/go/traits")
		}

		removeUnusedImports(node)

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, node); err != nil {
			return 0, fmt.Errorf("formatting AST: %w", err)
		}

		formatted, err := format.Source(buf.Bytes())
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

type packageWrapInfo struct {
	importPath string
	pkgName    string
}

type wrapResult struct {
	targetPkg         string
	clientConstructor string
	serviceDesc       string
	needsTraitsImport bool
}

func getWrapInfo(pkgInfo *packageWrapInfo, funcName string) *wrapResult {
	if !strings.HasPrefix(funcName, "Wrap") {
		return nil
	}

	if isSimpleWrapFunc(funcName) {
		result := getTraitPackageWrapInfo(pkgInfo, funcName)
		if result != nil {
			return result
		}
	}

	if len(funcName) > 4 && funcName != "Wrap" {
		return getMultiTraitPackageWrapInfo(pkgInfo, funcName)
	}

	return nil
}

func isSimpleWrapFunc(funcName string) bool {
	aspect := strings.TrimPrefix(funcName, "Wrap")
	if aspect == "" || aspect == funcName {
		return false
	}

	aspects := []string{"Api", "Info", "History"}
	for _, validAspect := range aspects {
		if aspect == validAspect {
			return true
		}
	}

	return false
}

func getTraitPackageWrapInfo(pkgInfo *packageWrapInfo, funcName string) *wrapResult {
	traitInfo := getTraitInfoFromPackage(pkgInfo.importPath, pkgInfo.pkgName)
	if traitInfo == nil {
		return nil
	}

	aspect := strings.TrimPrefix(funcName, "Wrap")
	if aspect == "" {
		return nil
	}

	var targetPkg string
	var needsTraitsImport bool

	if strings.Contains(pkgInfo.importPath, "/sc-golang/pkg/trait/") {
		targetPkg = "traits"
		needsTraitsImport = true
	} else if pkgInfo.importPath == "github.com/smart-core-os/sc-api/go/traits" {
		targetPkg = pkgInfo.pkgName
		needsTraitsImport = false
	} else if strings.Contains(pkgInfo.importPath, "/sc-bos/pkg/gentrait/") {
		targetPkg = pkgInfo.pkgName
		needsTraitsImport = false
	} else {
		return nil
	}

	clientConstructor := "New" + traitInfo.traitName + aspect + "Client"
	serviceDesc := traitInfo.traitName + aspect + "_ServiceDesc"

	return &wrapResult{
		targetPkg:         targetPkg,
		clientConstructor: clientConstructor,
		serviceDesc:       serviceDesc,
		needsTraitsImport: needsTraitsImport,
	}
}

func getMultiTraitPackageWrapInfo(pkgInfo *packageWrapInfo, funcName string) *wrapResult {
	if !strings.HasPrefix(funcName, "Wrap") || funcName == "Wrap" {
		return nil
	}

	serviceName := strings.TrimPrefix(funcName, "Wrap")

	return &wrapResult{
		targetPkg:         pkgInfo.pkgName,
		clientConstructor: "New" + serviceName + "Client",
		serviceDesc:       serviceName + "_ServiceDesc",
		needsTraitsImport: false,
	}
}

type traitInfo struct {
	PackageName       string
	traitName         string
	ClientConstructor string
	ServiceDesc       string
	InfoConstructor   string
	InfoServiceDesc   string
}

func getTraitInfoFromPackage(importPath, pkgName string) *traitInfo {
	if !strings.Contains(importPath, "/sc-golang/pkg/trait/") &&
		!strings.Contains(importPath, "/sc-bos/pkg/gentrait/") &&
		importPath != "github.com/smart-core-os/sc-api/go/traits" {
		return nil
	}

	parts := strings.Split(importPath, "/")
	packageName := parts[len(parts)-1]

	if strings.HasSuffix(packageName, "pb") {
		packageName = strings.TrimSuffix(packageName, "pb")
	}

	traitName := toPascalCase(packageName)

	return &traitInfo{
		PackageName:       pkgName,
		traitName:         traitName,
		ClientConstructor: "New" + traitName + "ApiClient",
		ServiceDesc:       traitName + "Api_ServiceDesc",
		InfoConstructor:   "New" + traitName + "InfoClient",
		InfoServiceDesc:   traitName + "Info_ServiceDesc",
	}
}

var traitWords = buildWordsByFirstChar()

func buildWordsByFirstChar() map[byte]map[string]bool {
	var words = []string{
		"access", "air",
		"brightness", "booking", "button",
		"channel", "close", "color", "count",
		"emergency", "electric", "energy", "events", "extend", "enter", "event",
		"fluid", "flow", "fan",
		"hail",
		"input",
		"lighting", "leave", "light", "lock",
		"microphone", "metadata", "motion", "meter", "mode",
		"occupancy", "open", "off", "on",
		"publication", "pressure", "parent", "press", "ptz",
		"quality",
		"retract",
		"service", "speaker", "storage", "select", "sensor", "sound", "speed",
		"temperature", "ticket", "test",
		"unlock",
		"vending",
		"waste",
	}

	result := make(map[byte]map[string]bool)
	for _, w := range words {
		if len(w) == 0 {
			continue
		}
		firstChar := w[0]
		if result[firstChar] == nil {
			result[firstChar] = make(map[string]bool)
		}
		result[firstChar][w] = true
	}
	return result
}

func toPascalCase(s string) string {
	memo := make(map[int]string)

	var parse func(int) (string, bool)
	parse = func(start int) (string, bool) {
		if start == len(s) {
			return "", true
		}

		if result, found := memo[start]; found {
			return result, result != ""
		}

		firstChar := s[start]
		wordSet := traitWords[firstChar]
		if len(wordSet) == 0 {
			memo[start] = ""
			return "", false
		}

		for end := start + 1; end <= len(s); end++ {
			word := s[start:end]
			if wordSet[word] {
				rest, ok := parse(end)
				if ok {
					result := strings.ToUpper(word[:1]) + word[1:] + rest
					memo[start] = result
					return result, true
				}
			}
		}

		memo[start] = ""
		return "", false
	}

	result, ok := parse(0)
	if ok {
		return result
	}

	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

func ensureImport(file *ast.File, importPath string) {
	for _, imp := range file.Imports {
		if imp.Path != nil && strings.Trim(imp.Path.Value, `"`) == importPath {
			return
		}
	}

	newImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf(`"%s"`, importPath),
		},
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		genDecl.Specs = append(genDecl.Specs, newImport)
		return
	}

	file.Decls = append([]ast.Decl{
		&ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: []ast.Spec{newImport},
		},
	}, file.Decls...)
}

func removeUnusedImports(file *ast.File) {
	usedPkgs := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			return false
		}

		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				usedPkgs[ident.Name] = true
			}
		}
		return true
	})

	for i := 0; i < len(file.Decls); i++ {
		genDecl, ok := file.Decls[i].(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		newSpecs := []ast.Spec{}
		for _, spec := range genDecl.Specs {
			impSpec, ok := spec.(*ast.ImportSpec)
			if !ok || impSpec.Path == nil {
				newSpecs = append(newSpecs, spec)
				continue
			}

			importPath := strings.Trim(impSpec.Path.Value, `"`)

			var pkgName string
			if impSpec.Name != nil {
				pkgName = impSpec.Name.Name
			} else {
				parts := strings.Split(importPath, "/")
				pkgName = parts[len(parts)-1]
			}

			if impSpec.Name != nil && (impSpec.Name.Name == "_" || impSpec.Name.Name == ".") {
				newSpecs = append(newSpecs, spec)
				continue
			}

			if usedPkgs[pkgName] {
				newSpecs = append(newSpecs, spec)
			}
		}

		if len(newSpecs) == 0 {
			file.Decls = append(file.Decls[:i], file.Decls[i+1:]...)
			i--
		} else {
			genDecl.Specs = newSpecs
		}
	}
}
