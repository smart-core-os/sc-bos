// Package uiproto generates JavaScript and TypeScript code from Protocol Buffer definitions.
package uiproto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/generator"
	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/protofile"
	"github.com/smart-core-os/sc-bos/cmd/tools/genproto/internal/toolchain"
)

var Step = generator.Step{
	ID:   "uiproto",
	Desc: "UI protoc code generation",
	Run:  run,
}

func run(ctx *generator.Context) error {
	protoDir := filepath.Join(ctx.RootDir, "proto")
	uiGenDir := filepath.Join(ctx.RootDir, "ui", "ui-gen")
	outDir := filepath.Join(uiGenDir, "proto")

	// Ensure output directory exists
	if !ctx.DryRun {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	// Discover proto files
	protoFiles, err := protofile.Discover(protoDir)
	if err != nil {
		return fmt.Errorf("discovering proto files: %w", err)
	}
	ctx.Verbose("Found %d proto files", len(protoFiles))

	// Generate protobuf code
	if err := generateProtos(ctx, protoDir, outDir, protoFiles); err != nil {
		return err
	}

	// Fix generated files
	if err := fixGeneratedFiles(ctx, outDir); err != nil {
		return err
	}

	return nil
}

// generateProtos generates JavaScript and TypeScript code from proto files.
func generateProtos(ctx *generator.Context, protoDir, outDir string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	ctx.Verbose("Generating JS/TS code for %d files", len(files))

	args := []string{"protoc", "--", "-I", protoDir}
	args = append(args,
		"--js_out=import_style=commonjs:"+outDir,
		"--grpc-web_out=import_style=commonjs+dts,mode=grpcwebtext:"+outDir,
	)
	args = append(args, files...)

	if ctx.DryRun {
		ctx.Info("[DRY RUN] Would run: protomod %s", strings.Join(args, " "))
		return nil
	}

	return toolchain.RunProtomod(protoDir, args...)
}

// fixGeneratedFiles applies import path fixes to generated JavaScript and TypeScript files.
func fixGeneratedFiles(ctx *generator.Context, outDir string) error {
	ctx.Verbose("Fixing generated files in %s", outDir)

	// Find all _pb.js files
	jsFiles, err := filepath.Glob(filepath.Join(outDir, "*_pb.js"))
	if err != nil {
		return fmt.Errorf("finding _pb.js files: %w", err)
	}

	for _, file := range jsFiles {
		if err := fixJSFile(ctx, file); err != nil {
			return fmt.Errorf("fixing %s: %w", filepath.Base(file), err)
		}
	}

	// Find all _pb.d.ts files
	dtsFiles, err := filepath.Glob(filepath.Join(outDir, "*_pb.d.ts"))
	if err != nil {
		return fmt.Errorf("finding _pb.d.ts files: %w", err)
	}

	for _, file := range dtsFiles {
		if err := fixDTSFile(ctx, file); err != nil {
			return fmt.Errorf("fixing %s: %w", filepath.Base(file), err)
		}
	}

	return nil
}

// fixJSFile replaces relative imports with package imports in JavaScript files.
func fixJSFile(ctx *generator.Context, filePath string) error {
	if ctx.DryRun {
		ctx.Debug("[DRY RUN] Would fix imports in %s", filepath.Base(filePath))
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	fixed := fixJSImports(content)

	if err := os.WriteFile(filePath, fixed, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// fixDTSFile replaces relative imports with package imports in TypeScript definition files.
func fixDTSFile(ctx *generator.Context, filePath string) error {
	if ctx.DryRun {
		ctx.Debug("[DRY RUN] Would fix imports in %s", filepath.Base(filePath))
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	fixed := fixDTSImports(content)

	if err := os.WriteFile(filePath, fixed, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
