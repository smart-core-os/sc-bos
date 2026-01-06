// Package goproto generates Go code from Protocol Buffer definitions.
//
// Proto files with services are generated with wrapper support.
// Files where all service rpc requests have a `string name` field are generated with router support.
package goproto

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	for _, dir := range []string{
		filepath.Join(ctx.RootDir, "pkg", "gen"),   // generated code used to go here
		filepath.Join(ctx.RootDir, "pkg", "proto"), // current location for generated code
	} {
		// Clean up old generated files
		if err := cleanGeneratedFiles(ctx, dir); err != nil {
			return fmt.Errorf("cleaning generated files %q: %w", dir, err)
		}
	}

	// Discover proto files and their required generators
	fileGenerators, err := analyzeProtoFiles(protoDir)
	if err != nil {
		return fmt.Errorf("analyzing proto files: %w", err)
	}
	groups := groupByGeneratorSet(fileGenerators)
	ctx.Verbose("Found %d proto files in %d generator groups", len(fileGenerators), len(groups))

	for gen, files := range groups {
		if err := generateProtos(ctx, protoDir, gen, files); err != nil {
			return err
		}
	}

	return nil
}

// groupByGeneratorSet groups proto files by their generator flags.
func groupByGeneratorSet(fileGenerators map[string]Generator) map[Generator][]string {
	buckets := make(map[Generator][]string)
	for file, gen := range fileGenerators {
		buckets[gen] = append(buckets[gen], file)
	}
	// Sort each bucket for deterministic output
	for _, files := range buckets {
		slices.Sort(files)
	}
	return buckets
}

// cleanGeneratedFiles removes old generated files from the output directory.
func cleanGeneratedFiles(ctx *generator.Context, genDir string) error {
	ctx.Verbose("Cleaning old generated files from %s", genDir)

	if ctx.DryRun {
		ctx.Info("[DRY RUN] Would remove old generated files matching .pb.go")
		return nil
	}

	if _, err := os.Stat(genDir); os.IsNotExist(err) {
		// Directory doesn't exist yet, nothing to clean
		return nil
	}

	removed := 0
	emptyDirs := []string{}

	err := filepath.WalkDir(genDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Remove any .pb.go files (matches *.pb.go, *_grpc.pb.go, *_router.pb.go, *_wrap.pb.go, etc.)
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

	// Clean up empty directories (walk in reverse to handle nested directories)
	if removed > 0 {
		err = filepath.WalkDir(genDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() || path == genDir {
				return nil
			}
			emptyDirs = append(emptyDirs, path)
			return nil
		})
		if err != nil {
			return fmt.Errorf("collecting directories: %w", err)
		}

		// Remove directories in reverse order (deepest first)
		for i := len(emptyDirs) - 1; i >= 0; i-- {
			dir := emptyDirs[i]
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			if len(entries) == 0 {
				if err := os.Remove(dir); err == nil {
					ctx.Debug("Removed empty directory %s", dir)
				}
			}
		}

		ctx.Verbose("Removed %d old generated file(s)", removed)
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
