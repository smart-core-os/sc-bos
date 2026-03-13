package scgolang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

// Fix rewrites github.com/smart-core-os/sc-golang import paths to their new
// locations in github.com/smart-core-os/sc-bos.
var Fix = fixer.Fix{
	ID:   "scgolangmv",
	Desc: "Update imports from github.com/smart-core-os/sc-golang to their new locations in sc-bos",
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
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	// Quick check before parsing.
	if !strings.Contains(string(content), oldModule) {
		return 0, nil
	}

	newContent, changed, err := transformImports(string(content))
	if err != nil {
		return 0, err
	}

	if !changed {
		return 0, nil
	}

	if !ctx.DryRun {
		if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
			return 0, fmt.Errorf("writing file: %w", err)
		}
	}

	ctx.Verbose("  Modified %s", relPath(ctx.RootDir, filename))
	return 1, nil
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
