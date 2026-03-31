// Package regoimports updates Rego data references when proto packages are renamed.
//
// It detects renames by comparing each .rego file's name (which reflects the old
// proto package path) against its package declaration (the new path), then
// applies those renames as text replacements across all .rego files.
package regoimports

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "regoimports",
	Desc: "Update Rego data references for proto package renames",
	Run:  run,
}

func ShouldProcessFile(path string) bool {
	if !strings.HasSuffix(path, ".rego") {
		return false
	}

	if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
		return false
	}

	if strings.Contains(path, "/cmd/tools/scfix/internal/") && strings.Contains(path, "/testdata/") {
		return false
	}

	return true
}

func run(ctx *fixer.Context) (int, error) {
	renames, err := buildRenameMap(ctx.RootDir)
	if err != nil {
		return 0, fmt.Errorf("building rename map: %w", err)
	}

	if len(renames) == 0 {
		ctx.Verbose("No rego package renames found")
		return 0, nil
	}

	ctx.Verbose("Found %d rego package rename(s):", len(renames))
	for old, new := range renames {
		ctx.Verbose("  %s -> %s", old, new)
	}

	totalChanges := 0
	err = filepath.WalkDir(ctx.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !ShouldProcessFile(path) {
			return nil
		}
		changes, err := processFile(ctx, path, renames)
		if err != nil {
			return fmt.Errorf("processing %s: %w", path, err)
		}
		totalChanges += changes
		return nil
	})
	return totalChanges, err
}

// buildRenameMap scans rego files to find renames.
// A rename is detected when the file name (without .rego) contains a proto version segment (e.g. ".v1.")
// but the package declaration does not — meaning the package was renamed to drop the version.
func buildRenameMap(rootDir string) (map[string]string, error) {
	renames := make(map[string]string)
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !ShouldProcessFile(path) {
			return err
		}
		oldPkg := strings.TrimSuffix(filepath.Base(path), ".rego")
		newPkg, err := readPackageDecl(path)
		if err != nil || newPkg == "" || newPkg == oldPkg {
			return err
		}
		if hasVersionSegment(oldPkg) && !hasVersionSegment(newPkg) {
			renames[newPkg] = oldPkg
		}
		return nil
	})
	return renames, err
}

// hasVersionSegment reports whether s contains a proto version segment like "v1" or "v1alpha".
func hasVersionSegment(s string) bool {
	for part := range strings.SplitSeq(s, ".") {
		if len(part) >= 2 && part[0] == 'v' && part[1] >= '1' && part[1] <= '9' {
			return true
		}
	}
	return false
}

func readPackageDecl(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if after, ok := strings.CutPrefix(line, "package "); ok {
			return after, nil
		}
	}
	return "", scanner.Err()
}

func processFile(ctx *fixer.Context, path string, renames map[string]string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	newContent := string(content)
	changes := 0
	for oldPkg, newPkg := range renames {
		updated := strings.ReplaceAll(newContent, oldPkg, newPkg)
		if updated != newContent {
			changes++
			newContent = updated
		}
	}

	if changes == 0 {
		return 0, nil
	}

	rel := relPath(ctx.RootDir, path)
	if ctx.DryRun {
		ctx.Info("  Would update: %s", rel)
		return changes, nil
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return 0, fmt.Errorf("writing file: %w", err)
	}

	ctx.Info("  Updated: %s (%d package reference(s))", rel, changes)
	return changes, nil
}

func relPath(rootDir, path string) string {
	if rel, err := filepath.Rel(rootDir, path); err == nil {
		return rel
	}
	return filepath.Base(path)
}
