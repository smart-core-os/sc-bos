// Command genprototypemap generates the type-to-trait mapping for the goprotoimports fixer.
// It scans pkg/proto/*pb directories to build a map of all exported symbols to their trait packages.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Find the repository root
	rootDir, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("finding repo root: %w", err)
	}

	// Scan generated Go files and extract exported symbols
	symbolToTrait := make(map[string]string)

	// Scan pkg/proto/*pb (after protogopkg fixer and genproto have run)
	protoDir := filepath.Join(rootDir, "pkg", "proto")
	if _, err := os.Stat(protoDir); err != nil {
		return fmt.Errorf("pkg/proto directory not found - run protogopkg fixer and go generate first: %w", err)
	}

	err = filepath.WalkDir(protoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root proto directory itself
		if path == protoDir {
			return nil
		}

		// Look for trait package directories (e.g., meterpb, hubpb, driver/dalipb)
		if d.IsDir() && strings.HasSuffix(d.Name(), "pb") {
			traitPkg := d.Name()
			// Get the relative path from pkg/proto for the import path
			relPath, err := filepath.Rel(protoDir, path)
			if err != nil {
				return fmt.Errorf("getting relative path: %w", err)
			}
			// Convert to forward slashes for import paths
			importPath := filepath.ToSlash(relPath)

			if err := scanDirectory(path, symbolToTrait, traitPkg, importPath); err != nil {
				return fmt.Errorf("scanning %s: %w", path, err)
			}
			// Skip descending into this directory since scanDirectory already walked it
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("scanning pkg/proto: %w", err)
	}

	if len(symbolToTrait) == 0 {
		return fmt.Errorf("no symbols found - ensure proto files have been generated in pkg/proto/")
	}

	// Generate the Go file
	if err := generateFile(symbolToTrait); err != nil {
		return fmt.Errorf("generating file: %w", err)
	}

	fmt.Printf("Generated type map with %d entries\n", len(symbolToTrait))
	return nil
}

// scanDirectory scans a directory for .pb.go files and extracts symbols.
// The traitPkg parameter is the package name (e.g., "meterpb", "hubpb", "dalipb").
// The importPath parameter is the relative path from pkg/proto (e.g., "meterpb", "driver/dalipb").
// It maps old gen package symbol names to new import paths with symbols (e.g., "meterpb.WrapApi", "driver/dalipb.AddToGroupRequest").
func scanDirectory(dir string, symbolToTrait map[string]string, traitPkg, importPath string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".pb.go") {
			return nil
		}

		// Extract the base filename to derive old symbol names
		basename := filepath.Base(path)
		basename = strings.TrimSuffix(basename, ".pb.go")

		// Parse the Go file
		symbols, err := extractExportedSymbols(path)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		for _, symbol := range symbols {
			// Derive the old gen package symbol name and map it to new importPath.symbol
			// For example: "WrapMeterApi" → "meterpb.WrapApi" or "AddToGroupRequest" → "driver/dalipb.AddToGroupRequest"
			oldSymbol := deriveOldGenSymbol(symbol, basename, traitPkg)
			newSymbol := importPath + "." + symbol

			// Use lowercase key to make lookups case-insensitive
			// This avoids camel-casing issues with multi-word traits
			key := strings.ToLower(oldSymbol)

			// Only add if not already present
			if _, exists := symbolToTrait[key]; !exists {
				symbolToTrait[key] = newSymbol
			}
		}

		return nil
	})
}

// deriveOldGenSymbol derives what the symbol name was in the old gen package.
// In the gen package, symbols often included the trait name to avoid collisions.
// For example:
//   - File: meter_api_wrap.pb.go, Symbol: WrapApi → Old: WrapMeterApi
//   - File: meter_api_router.pb.go, Symbol: NewApiRouter → Old: NewMeterApiRouter
//   - File: meter.pb.go, Symbol: MeterReading → Old: MeterReading (no change)
func deriveOldGenSymbol(symbol, basename, traitPkg string) string {
	// Strip the "pb" suffix from package name to get trait name
	trait := strings.TrimSuffix(traitPkg, "pb")

	// Convert trait name to title case for insertion into symbol name
	// e.g., "meter" → "Meter", "emergencylight" → "Emergencylight"
	traitTitle := strings.Title(trait)

	// Check if this file is a wrapper, router, or other specialized file
	// For these files, the trait name was inserted into the middle of the function name
	// e.g., WrapApi → WrapMeterApi, NewApiRouter → NewMeterApiRouter

	// Handle Wrap functions: WrapApi, WrapInfo, WrapHistory, WrapAdminApi
	if strings.HasPrefix(symbol, "Wrap") && !strings.HasPrefix(symbol, "Wrap"+traitTitle) {
		// Insert trait after "Wrap": WrapApi → WrapMeterApi
		return "Wrap" + traitTitle + strings.TrimPrefix(symbol, "Wrap")
	}

	// Handle New*Router functions: NewApiRouter, NewHistoryRouter, NewInfoRouter
	if strings.HasPrefix(symbol, "New") && strings.HasSuffix(symbol, "Router") && !strings.Contains(symbol, traitTitle) {
		// Insert trait after "New": NewApiRouter → NewMeterApiRouter
		middle := strings.TrimPrefix(strings.TrimSuffix(symbol, "Router"), "New")
		return "New" + traitTitle + middle + "Router"
	}

	// Symbol already includes trait or doesn't need it (e.g., message types, services)
	return symbol
}

// extractExportedSymbols parses a Go file and extracts all exported identifiers
// from top-level declarations (type, var, const, func). An identifier is exported if
// its first letter is uppercase. Only extracts functions, not methods.
func extractExportedSymbols(path string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var symbols []string

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// Handle type, var, const declarations
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					// Type declarations: type Foo struct{...}
					if s.Name.IsExported() {
						symbols = append(symbols, s.Name.Name)
					}
				case *ast.ValueSpec:
					// Var and const declarations
					for _, name := range s.Names {
						if name.IsExported() {
							symbols = append(symbols, name.Name)
						}
					}
				}
			}
		case *ast.FuncDecl:
			// Function declarations (but not methods)
			// Methods have a receiver, functions don't
			if d.Recv == nil && d.Name.IsExported() {
				symbols = append(symbols, d.Name.Name)
			}
		}
	}

	return symbols, nil
}

func generateFile(symbolToTrait map[string]string) error {
	var buf bytes.Buffer

	buf.WriteString(`// Code generated by go generate; DO NOT EDIT.

package goprotoimports

// typeToTraitMap maps exported symbols from generated proto code to their trait package names.
// This includes message types, service types, nested types, enums, generated helpers, and functions.
// Generated by parsing Go files in pkg/proto/*pb directories.
var typeToTraitMap = map[string]string{
`)

	// Sort keys for deterministic output
	var keys []string
	for k := range symbolToTrait {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Fprintf(&buf, "\t%q: %q,\n", k, symbolToTrait[k])
	}

	buf.WriteString("}\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting generated code: %w", err)
	}

	// Write to file in the parent package directory
	// When run via go generate, the working directory is the package directory
	outputPath := "typemap_generated.go"
	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func findRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
