// Package nodeclient replaces deprecated node.Node.Client calls with direct client constructors using ClientConn.
//
// Example transformation:
//
//	var client traits.OnOffApiClient
//	err := n.Client(&client)
//	if err != nil {
//	    return err
//	}
//	=>
//	client := traits.NewOnOffApiClient(n.ClientConn())
package nodeclient

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
	ID:   "nodeclient",
	Desc: "Replace node.Node.Client with direct client constructors using ClientConn",
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
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return 0, err
	}

	replacements := findReplacements(file)
	if len(replacements) == 0 {
		return 0, nil
	}

	logReplacements(ctx, replacements, filename)
	applyReplacements(replacements)

	if !ctx.DryRun {
		if err := writeFormattedFile(fset, file, filename); err != nil {
			return 0, err
		}
	}

	changes := len(replacements)
	ctx.Verbose("  Modified %s (%d changes)", filepath.Base(filename), changes)
	return changes, nil
}

func logReplacements(ctx *fixer.Context, replacements []replacement, filename string) {
	for _, repl := range replacements {
		ctx.Verbose("  Found n.Client(&%s) in %s", exprToString(repl.clientRef), filepath.Base(filename))
	}
}

func applyReplacements(replacements []replacement) {
	// Apply in reverse order to avoid index shifting issues
	for i := len(replacements) - 1; i >= 0; i-- {
		applyReplacement(replacements[i])
	}
}

func writeFormattedFile(fset *token.FileSet, file *ast.File, filename string) error {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, file); err != nil {
		return fmt.Errorf("formatting AST: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting source: %w", err)
	}

	if err := os.WriteFile(filename, formatted, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
