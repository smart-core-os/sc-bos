// Package optclients replaces deprecated node.WithOptClients with node.WithClients
// and node.HasOptClient with node.HasClient.
package optclients

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

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "optclients",
	Desc: "Replace deprecated node.WithOptClients and node.HasOptClient with their counterparts",
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

	hasNodeImport := false
	for _, imp := range node.Imports {
		if imp.Path.Value == `"github.com/smart-core-os/sc-bos/pkg/node"` {
			hasNodeImport = true
			break
		}
	}

	if !hasNodeImport {
		return 0, nil
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
		if !ok || pkgIdent.Name != "node" {
			return true
		}

		funcName := selExpr.Sel.Name
		var newFuncName string

		switch funcName {
		case "WithOptClients":
			newFuncName = "WithClients"
		case "HasOptClient":
			newFuncName = "HasClient"
		default:
			return true
		}

		ctx.Verbose("  Found node.%s in %s", funcName, filepath.Base(filename))

		selExpr.Sel.Name = newFuncName
		modified = true
		changes++

		return true
	})

	if !modified {
		return 0, nil
	}

	if !ctx.DryRun {
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
