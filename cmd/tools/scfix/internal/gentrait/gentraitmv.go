package gentrait

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var FixMove = fixer.Fix{
	ID:   "gentraitmv",
	Desc: "Move Go files from pkg/gentrait to pkg/proto",
	Run:  runMove,
}

func runMove(ctx *fixer.Context) (int, error) {
	totalChanges := 0

	gentraitDir := filepath.Join(ctx.RootDir, "pkg", "gentrait")
	if _, err := os.Stat(gentraitDir); os.IsNotExist(err) {
		return 0, nil
	}

	err := filepath.WalkDir(gentraitDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !fixer.ShouldProcessFileDir(path, d) || strings.HasSuffix(path, ".pb.go") {
			return nil
		}

		changes, err := moveFile(ctx, path, gentraitDir)
		if err != nil {
			relPath, _ := filepath.Rel(ctx.RootDir, path)
			return fmt.Errorf("moving %s: %w", relPath, err)
		}
		totalChanges += changes
		return nil
	})

	return totalChanges, err
}

func moveFile(ctx *fixer.Context, srcPath, gentraitDir string) (int, error) {
	relPath, err := filepath.Rel(gentraitDir, srcPath)
	if err != nil {
		return 0, fmt.Errorf("calculating relative path: %w", err)
	}

	destPath, err := gentraitToProtoFilePath(ctx.RootDir, relPath)
	if err != nil {
		ctx.Verbose("  Skipping %s: %v", relPath, err)
		return 0, nil
	}

	content, err := os.ReadFile(srcPath)
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", relPath, err)
	}

	transformedContent, err := transformFileContent(string(content), relPath)
	if err != nil {
		return 0, fmt.Errorf("transforming %s: %w", relPath, err)
	}

	relDestPath, _ := filepath.Rel(ctx.RootDir, destPath)
	ctx.Verbose("  Moving %s -> %s", relPath, relDestPath)

	if !ctx.DryRun {
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return 0, fmt.Errorf("creating directory for %s: %w", relDestPath, err)
		}

		if err := os.WriteFile(destPath, []byte(transformedContent), 0644); err != nil {
			return 0, fmt.Errorf("writing %s: %w", relDestPath, err)
		}

		if err := os.Remove(srcPath); err != nil {
			return 0, fmt.Errorf("removing %s: %w", relPath, err)
		}
	}

	return 1, nil
}
