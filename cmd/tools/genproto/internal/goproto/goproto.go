// Package goproto generates Go code from Protocol Buffer definitions.
//
// Proto files with services are generated with wrapper support.
// Files where all service rpc requests have a `string name` field are generated with router support.
package goproto

import (
	"fmt"
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
	genDir := filepath.Join(ctx.RootDir, "pkg", "gen")
	rootDir := ctx.RootDir

	// Discover proto files and their required generators
	fileGenerators, err := analyzeProtoFiles(protoDir)
	if err != nil {
		return fmt.Errorf("analyzing proto files: %w", err)
	}
	groups := groupByGeneratorSet(fileGenerators)
	ctx.Verbose("Found %d proto files in %d generator groups", len(fileGenerators), len(groups))

	for gen, files := range groups {
		if err := generateProtos(ctx, protoDir, genDir, rootDir, gen, files); err != nil {
			return err
		}
	}

	if err := fixGeneratedFiles(ctx, genDir); err != nil {
		return err
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

// generateProtos generates code for a set of proto files with the same generator requirements.
func generateProtos(ctx *generator.Context, protoDir, genDir, rootDir string, gen Generator, files []string) error {
	if len(files) == 0 {
		return nil
	}

	ctx.Verbose("Generating %s: %s", gen, strings.Join(files, ", "))

	args := []string{"protoc", "--", "-I", protoDir}
	args = append(args,
		"--go_out=paths=source_relative:"+genDir,
		"--go-grpc_out=paths=source_relative:"+genDir,
	)

	if gen.Has(GenRouter) {
		args = append(args, "--router_out="+rootDir)
	}
	if gen.Has(GenWrapper) {
		args = append(args, "--wrapper_out="+rootDir)
	}

	args = append(args, files...)

	if ctx.DryRun {
		ctx.Info("[DRY RUN] Would run: protomod %s", strings.Join(args, " "))
		return nil
	}

	return toolchain.RunProtomod(protoDir, args...)
}
