// Package goproto generates Go code from Protocol Buffer definitions.
//
// Proto files with services are generated with wrapper support.
// Files where all service rpc requests have a `string name` field are generated with router support.
package goproto

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/generator"
	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/toolchain"
)

var Step = generator.Step{
	ID:   "goproto",
	Desc: "Go protoc code generation",
	Run:  run,
}

// Generator represents protoc generator flags using a bitset.
type Generator uint8

const (
	GenRouter  Generator = 1 << iota // Generates router code for routed APIs
	GenWrapper                       // Generates wrapper code for services
)

// Has checks if a generator is enabled.
func (g Generator) Has(flag Generator) bool {
	return g&flag != 0
}

// String returns a human-readable description of enabled generators.
func (g Generator) String() string {
	if g == 0 {
		return "basic"
	}
	var parts []string
	if g.Has(GenRouter) {
		parts = append(parts, "router")
	}
	if g.Has(GenWrapper) {
		parts = append(parts, "wrapper")
	}
	return strings.Join(parts, "+")
}

func run(ctx *generator.Context) error {
	protoDir := filepath.Join(ctx.RootDir, "proto")

	// Discover proto files and their required generators.
	// This must happen before cleaning so we know which dirs are bos-owned.
	fileInfos, err := analyzeProtoFiles(protoDir)
	if err != nil {
		return fmt.Errorf("analyzing proto files: %w", err)
	}

	ownedDirs := collectOwnedDirs(ctx, fileInfos)
	if err := cleanGeneratedFiles(ctx, ownedDirs); err != nil {
		return fmt.Errorf("cleaning generated files: %w", err)
	}

	groups := groupByGeneratorSet(fileInfos)
	ctx.Verbose("Found %d proto files in %d generator groups", len(fileInfos), len(groups))

	for gen, files := range groups {
		if err := generateProtos(ctx, protoDir, gen, files); err != nil {
			return err
		}
	}

	return nil
}

// collectOwnedDirs returns the set of repo-relative output dirs owned by bos proto files.
// A dir is owned if at least one bos proto file outputs to it (i.e. OutputDir is non-empty).
func collectOwnedDirs(ctx *generator.Context, fileInfos map[string]ProtoFileInfo) map[string]bool {
	dirs := make(map[string]bool)
	for _, info := range fileInfos {
		if info.OutputDir != "" {
			dirs[info.OutputDir] = true
		}
	}
	ctx.Verbose("Identified %d owned output dirs", len(dirs))
	return dirs
}

// cleanGeneratedFiles removes stale generated files.
// pkg/gen/ is cleaned entirely (legacy location, should be empty).
// Each owned dir is cleaned selectively, preserving files not derived from bos protos.
func cleanGeneratedFiles(ctx *generator.Context, ownedDirs map[string]bool) error {
	if ctx.DryRun {
		ctx.Info("[DRY RUN] Would remove old generated files matching .pb.go")
		return nil
	}

	// Clean pkg/gen/ entirely (legacy location).
	genDir := filepath.Join(ctx.RootDir, "pkg", "gen")
	if err := cleanAllPbGo(ctx, genDir); err != nil {
		return fmt.Errorf("cleaning legacy gen dir: %w", err)
	}

	// Selectively clean each bos-owned output dir.
	protoDir := filepath.Join(ctx.RootDir, "pkg", "proto")
	totalRemoved := 0
	for dir := range ownedDirs {
		fullDir := filepath.Join(ctx.RootDir, dir)
		n, err := cleanOwnedDir(ctx, fullDir)
		if err != nil {
			return fmt.Errorf("cleaning %s: %w", dir, err)
		}
		totalRemoved += n
	}

	if totalRemoved > 0 {
		ctx.Verbose("Removed %d old generated file(s)", totalRemoved)
	}

	// Remove empty subdirectories left behind in pkg/proto/.
	if totalRemoved > 0 {
		if err := removeEmptyDirs(ctx, protoDir); err != nil {
			return fmt.Errorf("removing empty dirs: %w", err)
		}
	}

	return nil
}

// cleanAllPbGo removes all .pb.go files under dir, then removes empty subdirectories.
// Used for the legacy pkg/gen/ location.
func cleanAllPbGo(ctx *generator.Context, dir string) error {
	ctx.Verbose("Cleaning old generated files from %s", dir)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	removed := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".pb.go") {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", d.Name(), err)
			}
			ctx.Debug("Removed %s", path)
			removed++
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory tree: %w", err)
	}

	if removed > 0 {
		ctx.Verbose("Removed %d old generated file(s) from %s", removed, dir)
		if err := removeEmptyDirs(ctx, dir); err != nil {
			return fmt.Errorf("removing empty dirs in %s: %w", dir, err)
		}
	}

	return nil
}

// cleanOwnedDir selectively removes generated files from a single bos-owned package dir.
// Only files that end in _router.pb.go / _wrap.pb.go, or whose header identifies them
// as derived from a bos proto (// source: smartcore/bos/...), are deleted.
// Returns the number of files removed.
func cleanOwnedDir(ctx *generator.Context, dir string) (int, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return 0, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("reading dir: %w", err)
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".pb.go") {
			continue
		}
		path := filepath.Join(dir, name)
		if shouldDeleteInOwnedDir(path, name) {
			if err := os.Remove(path); err != nil {
				return removed, fmt.Errorf("removing %s: %w", name, err)
			}
			ctx.Debug("Removed %s", path)
			removed++
		}
	}

	return removed, nil
}

// shouldDeleteInOwnedDir reports whether a .pb.go file in a bos-owned dir should be deleted.
// Router/wrapper files are always deleted (they have no // source: header).
// Other files are deleted only if their header identifies them as bos-derived.
func shouldDeleteInOwnedDir(path, name string) bool {
	if strings.HasSuffix(name, "_router.pb.go") || strings.HasSuffix(name, "_wrap.pb.go") {
		return true
	}
	return hasBosSourceHeader(path)
}

// hasBosSourceHeader reports whether the file's first 10 lines contain a
// "// source: smartcore/bos/..." comment, identifying it as generated from a bos proto.
func hasBosSourceHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for i := 0; i < 10 && scanner.Scan(); i++ {
		if strings.HasPrefix(scanner.Text(), "// source: smartcore/bos/") {
			return true
		}
	}
	return false
}

// removeEmptyDirs removes empty subdirectories within dir, deepest first.
func removeEmptyDirs(ctx *generator.Context, dir string) error {
	var dirs []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != dir {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for i := len(dirs) - 1; i >= 0; i-- {
		entries, err := os.ReadDir(dirs[i])
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			if err := os.Remove(dirs[i]); err == nil {
				ctx.Debug("Removed empty directory %s", dirs[i])
			}
		}
	}
	return nil
}

// generateProtos generates code for a set of proto files with the same generator requirements.
func generateProtos(ctx *generator.Context, protoDir string, gen Generator, files []string) error {
	if len(files) == 0 {
		return nil
	}
	modulePrefix := "github.com/smart-core-os/sc-bos"
	outDir := ctx.RootDir

	ctx.Verbose("Generating %s: %s", gen, strings.Join(files, ", "))
	goPluginPath, err := toolchain.GetGoToolPath("protoc-gen-go")
	if err != nil {
		return fmt.Errorf("getting protoc-gen-go path: %w", err)
	}
	ctx.Verbose("  protoc-gen-go path: %q", goPluginPath)
	grpcPluginPath, err := toolchain.GetGoToolPath("protoc-gen-go-grpc")
	if err != nil {
		return fmt.Errorf("getting protoc-gen-go-grpc path: %w", err)
	}
	ctx.Verbose("  protoc-gen-go-grpc path: %q", grpcPluginPath)

	args := []string{"protoc", "--", "-I", protoDir}
	args = append(args,
		"--plugin=protoc-gen-go="+goPluginPath,
		"--go_opt=module="+modulePrefix,
		"--go_out="+outDir,
		"--plugin=protoc-gen-go-grpc="+grpcPluginPath,
		"--go-grpc_opt=module="+modulePrefix,
		"--go-grpc_out="+outDir,
	)

	if gen.Has(GenRouter) {
		routerPluginPath, err := toolchain.GetGoToolPath("protoc-gen-router")
		if err != nil {
			return fmt.Errorf("getting protoc-gen-router path: %w", err)
		}
		ctx.Verbose("  protoc-gen-router path: %q", routerPluginPath)
		args = append(args,
			"--plugin=protoc-gen-router="+routerPluginPath,
			"--router_opt=usePaths=true",
			"--router_opt=module="+modulePrefix,
			"--router_out="+outDir,
		)
	}
	if gen.Has(GenWrapper) {
		wrapperPluginPath, err := toolchain.GetGoToolPath("protoc-gen-wrapper")
		if err != nil {
			return fmt.Errorf("getting protoc-gen-wrapper path: %w", err)
		}
		ctx.Verbose("  protoc-gen-wrapper path: %q", wrapperPluginPath)
		args = append(args,
			"--plugin=protoc-gen-wrapper="+wrapperPluginPath,
			"--wrapper_opt=usePaths=true",
			"--wrapper_opt=module="+modulePrefix,
			"--wrapper_out="+outDir,
		)
	}

	args = append(args, files...)

	if ctx.DryRun {
		ctx.Info("[DRY RUN] Would run: protomod %s", strings.Join(args, " "))
		return nil
	}

	return toolchain.RunProtomod(protoDir, args...)
}
