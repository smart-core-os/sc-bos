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

	// Clean up old generated files
	if err := cleanGeneratedFiles(ctx, outDir); err != nil {
		return fmt.Errorf("cleaning generated files: %w", err)
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

// cleanGeneratedFiles removes old generated files from the output directory.
func cleanGeneratedFiles(ctx *generator.Context, outDir string) error {
	ctx.Verbose("Cleaning old generated files from %s", outDir)

	if ctx.DryRun {
		ctx.Info("[DRY RUN] Would remove old generated files matching _pb.*")
		return nil
	}

	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		// Directory doesn't exist yet, nothing to clean
		return nil
	}

	removed := 0
	emptyDirs := []string{}

	err := filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Remove any _pb.* files (matches *_pb.js, *_pb.d.ts, *_grpc_web_pb.js, *_grpc_web_pb.d.ts, etc.)
		name := d.Name()
		if strings.Contains(name, "_pb.") {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", name, err)
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
		err = filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() || path == outDir {
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

	err := filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Check file extension and fix accordingly
		name := d.Name()
		if strings.HasSuffix(name, "_pb.js") {
			if err := fixJSFile(ctx, path); err != nil {
				return fmt.Errorf("fixing %s: %w", name, err)
			}
		} else if strings.HasSuffix(name, "_pb.d.ts") {
			if err := fixDTSFile(ctx, path); err != nil {
				return fmt.Errorf("fixing %s: %w", name, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("walking directory tree: %w", err)
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
