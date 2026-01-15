// Package protov1go updates Go code for service renames from protov1 fixer.
//
// When proto services are renamed (e.g., EnterLeaveHistory -> EnterLeaveSensorHistory),
// the generated Go code creates new symbols. This fixer updates existing code to use the new names.
package protov1go

// The generator must be run to populate serviceRenames so the tool can work against other repositories.

//go:generate go run ./genrenames -output service_renames_gen.go

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "protov1go",
	Desc: "Update Go code for service renames from protov1 fixer",
	Run:  run,
}

func run(ctx *fixer.Context) (int, error) {
	if len(serviceRenames) == 0 {
		ctx.Verbose("No service renames configured")
		return 0, nil
	}

	ctx.Verbose("Configured %d service rename(s):", len(serviceRenames))
	for oldName, newName := range serviceRenames {
		ctx.Verbose("  %s -> %s", oldName, newName)
	}

	totalChanges := 0

	// Process Go files in the codebase
	err := filepath.WalkDir(ctx.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !fixer.ShouldProcessFileDir(path, d) ||
			strings.HasSuffix(path, ".pb.go") {
			return nil
		}

		changed, err := processGoFile(ctx, path, serviceRenames)
		if err != nil {
			return fmt.Errorf("processing %s: %w", path, err)
		}
		if changed {
			totalChanges++
		}
		return nil
	})
	if err != nil {
		return totalChanges, err
	}

	return totalChanges, nil
}

func processGoFile(ctx *fixer.Context, path string, serviceRenames map[string]string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		// Skip files with parse errors
		ctx.Verbose("  Skipping %s: parse error: %v", relPath(ctx.RootDir, path), err)
		return false, nil
	}

	rewriter := &serviceRewriter{
		renames: serviceRenames,
		changed: false,
	}

	ast.Walk(rewriter, node)

	if !rewriter.changed {
		return false, nil
	}

	// Format the modified AST back to source
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return false, fmt.Errorf("formatting: %w", err)
	}

	newContent := buf.Bytes()

	rel := relPath(ctx.RootDir, path)
	if ctx.DryRun {
		ctx.Info("  Would update: %s", rel)
		return true, nil
	}

	if err := os.WriteFile(path, newContent, 0644); err != nil {
		return false, fmt.Errorf("writing file: %w", err)
	}

	ctx.Info("  Updated: %s", rel)
	return true, nil
}

type serviceRewriter struct {
	renames map[string]string
	changed bool
}

func (r *serviceRewriter) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.SelectorExpr:
		// Handle gen.XxxServer, gen.NewXxxClient, gen.RegisterXxxServer, etc.
		r.handleSelector(n)
	}
	return r
}

func (r *serviceRewriter) handleSelector(sel *ast.SelectorExpr) {
	// Check if selector is from "gen" package
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "gen" {
		return
	}

	oldName := sel.Sel.Name
	newName := r.tryRenameSymbol(oldName)

	if newName != "" && newName != oldName {
		sel.Sel.Name = newName
		r.changed = true
	}
}

func (r *serviceRewriter) tryRenameSymbol(symbol string) string {
	// Try various patterns of generated symbols

	for oldService, newService := range r.renames {
		patterns := []struct {
			prefix, suffix string
		}{
			// Server-side patterns
			{"Unimplemented", "Server"},
			{"Unsafe", "Server"},
			{"Register", "Server"},
			{"", "Server"},

			// Client-side patterns
			{"New", "Client"},
			{"", "Client"},

			// Wrapper patterns
			{"Wrap", ""},
			{"", "Wrapper"},

			// Router patterns
			{"New", "Router"},
			{"", "Router"},
			{"With", "ClientFactory"},

			// ServiceDesc pattern
			{"", "_ServiceDesc"},
		}

		for _, p := range patterns {
			oldPattern := p.prefix + oldService + p.suffix
			if symbol == oldPattern {
				return p.prefix + newService + p.suffix
			}
		}
	}

	return symbol
}

func relPath(rootDir, path string) string {
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}
