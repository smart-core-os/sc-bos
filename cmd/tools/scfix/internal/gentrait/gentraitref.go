package gentrait

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var FixRefs = fixer.Fix{
	ID:   "gentraitref",
	Desc: "Update references from pkg/gentrait to pkg/proto",
	Run:  runUpdateRefs,
}

func runUpdateRefs(ctx *fixer.Context) (int, error) {
	totalChanges := 0

	// Walk all Go files in the repository
	err := filepath.WalkDir(ctx.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !fixer.ShouldProcessFileDir(path, d) {
			return nil
		}

		// Skip files in pkg/gentrait (they were already handled by gentraitmv)
		relPath, _ := filepath.Rel(ctx.RootDir, path)
		if strings.HasPrefix(filepath.ToSlash(relPath), "pkg/gentrait/") {
			return nil
		}

		changes, err := updateFileReferences(ctx, path)
		if err != nil {
			return fmt.Errorf("updating %s: %w", relPath, err)
		}
		if changes > 0 {
			ctx.Verbose("  Updated %s", relPath)
		}
		totalChanges += changes
		return nil
	})

	return totalChanges, err
}

func updateFileReferences(ctx *fixer.Context, path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading file: %w", err)
	}

	originalContent := string(content)
	transformedContent, err := transformReferences(originalContent)
	if err != nil {
		return 0, fmt.Errorf("transforming references: %w", err)
	}

	// No changes needed
	if transformedContent == originalContent {
		return 0, nil
	}

	if !ctx.DryRun {
		if err := os.WriteFile(path, []byte(transformedContent), 0644); err != nil {
			return 0, fmt.Errorf("writing file: %w", err)
		}
	}

	return 1, nil
}
