package protov1js

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "protov1js",
	Desc: "Update JS/TS imports for proto files moved by protov1 fixer",
	Run:  run,
}

func run(ctx *fixer.Context) (int, error) {
	relProjectDirs, err := findProjectDirs(os.DirFS(ctx.RootDir))
	if err != nil {
		return 0, fmt.Errorf("finding projects: %w", err)
	}

	if len(relProjectDirs) == 0 {
		ctx.Verbose("No projects using @smart-core-os/sc-bos-ui-gen found")
		return 0, nil
	}

	ctx.Verbose("Found %d project(s) using @smart-core-os/sc-bos-ui-gen", len(relProjectDirs))

	totalChanges := 0
	for _, relProjectDir := range relProjectDirs {
		absProjectDir := filepath.Join(ctx.RootDir, filepath.FromSlash(relProjectDir))
		changes, err := processProject(ctx, absProjectDir)
		if err != nil {
			return totalChanges, fmt.Errorf("processing project %s: %w", relProjectDir, err)
		}
		totalChanges += changes
	}

	return totalChanges, nil
}
