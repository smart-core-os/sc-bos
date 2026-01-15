// Package protogopkg updates the Go package option in proto files from pkg/gen to pkg/proto/{trait}.
// This fixer should only be run against the sc-bos repository itself.
//
// Example transformations:
//
//	proto/smartcore/bos/meter/v1/meter.proto:
//	  option go_package = "github.com/smart-core-os/sc-bos/pkg/gen";
//	becomes:
//	  option go_package = "github.com/smart-core-os/sc-bos/pkg/proto/meterpb";
package protogopkg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "protogopkg",
	Desc: "Update Go package option in proto files from pkg/gen to pkg/proto/{trait}",
	Run:  run,
}

var (
	goPackagePattern = regexp.MustCompile(`^(\s*)option\s+go_package\s*=\s*"github\.com/smart-core-os/sc-bos/pkg/gen"\s*;?\s*$`)
	packagePattern   = regexp.MustCompile(`^\s*package\s+([\w.]+)\s*;?\s*$`)
)

func run(ctx *fixer.Context) (int, error) {
	totalChanges := 0

	protoDir := filepath.Join(ctx.RootDir, "proto")
	if _, err := os.Stat(protoDir); os.IsNotExist(err) {
		return 0, nil
	}

	err := filepath.WalkDir(protoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !shouldProcessProtoFile(path, d) {
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

func shouldProcessProtoFile(path string, d os.DirEntry) bool {
	if d.IsDir() {
		return false
	}

	if !strings.HasSuffix(path, ".proto") {
		return false
	}

	if strings.Contains(path, "/.git/") {
		return false
	}

	return true
}

func processFile(ctx *fixer.Context, filename string) (int, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	// Quick check: does file contain the old package path?
	if !strings.Contains(string(content), `"github.com/smart-core-os/sc-bos/pkg/gen"`) {
		return 0, nil
	}

	lines := strings.Split(string(content), "\n")

	// Extract the proto package from the file content
	protoPackage := extractProtoPackage(lines)
	if protoPackage == "" {
		ctx.Verbose("  Skipping %s: cannot extract proto package", relPath(ctx.RootDir, filename))
		return 0, nil
	}

	// Derive the new Go package from the proto package
	newPackage := deriveGoPackageFromProto(protoPackage)
	if newPackage == "" {
		ctx.Verbose("  Skipping %s: cannot derive Go package from proto package %s", relPath(ctx.RootDir, filename), protoPackage)
		return 0, nil
	}

	newLines, changes := processLines(ctx, filename, lines, newPackage)

	if changes == 0 {
		return 0, nil
	}

	if !ctx.DryRun {
		newContent := strings.Join(newLines, "\n")
		if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
			return 0, fmt.Errorf("writing file: %w", err)
		}
	}

	ctx.Verbose("  Modified %s (%d changes)", relPath(ctx.RootDir, filename), changes)

	return changes, nil
}

func processLines(ctx *fixer.Context, filename string, lines []string, newPackage string) ([]string, int) {
	var newLines []string
	var changes int

	for _, line := range lines {
		if match := goPackagePattern.FindStringSubmatch(line); match != nil {
			indent := match[1]
			newLine := fmt.Sprintf(`%soption go_package = "%s";`, indent, newPackage)
			newLines = append(newLines, newLine)
			changes++
			ctx.Verbose("  Replacing go_package in %s", relPath(ctx.RootDir, filename))
		} else {
			newLines = append(newLines, line)
		}
	}

	return newLines, changes
}

// extractProtoPackage extracts the package declaration from proto file lines.
// For example, "package smartcore.bos.meter.v1;" returns "smartcore.bos.meter.v1".
func extractProtoPackage(lines []string) string {
	for _, line := range lines {
		if match := packagePattern.FindStringSubmatch(line); match != nil {
			return match[1]
		}
	}
	return ""
}

// deriveGoPackageFromProto converts a proto package to the new Go package path.
// Only supports v1 packages. Returns empty string for unversioned or other versioned packages.
// Examples:
//
//	smartcore.bos.meter.v1 -> github.com/smart-core-os/sc-bos/pkg/proto/meterpb
//	smartcore.bos.driver.axiomxa.v1 -> github.com/smart-core-os/sc-bos/pkg/proto/driver/axiomxapb
//	smartcore.bos.airqualitysensor.v1 -> github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb
//	smartcore.bos.account -> "" (unversioned, not supported)
//	smartcore.bos.meter.v2 -> "" (v2, not supported)
func deriveGoPackageFromProto(protoPackage string) string {
	// Pattern: smartcore.bos.{path}.v1
	// We want to extract {path} and convert it to pkg/proto/{path}pb
	const prefix = "smartcore.bos."
	const versionSuffix = ".v1"

	if !strings.HasPrefix(protoPackage, prefix) {
		return ""
	}

	// Must end with .v1
	if !strings.HasSuffix(protoPackage, versionSuffix) {
		return ""
	}

	// Remove prefix and suffix to get the path
	remainder := strings.TrimPrefix(protoPackage, prefix)
	remainder = strings.TrimSuffix(remainder, versionSuffix)

	// Split by dots to handle nested packages
	parts := strings.Split(remainder, ".")

	if len(parts) == 0 {
		return ""
	}

	// Add "pb" suffix to the last component
	parts[len(parts)-1] = parts[len(parts)-1] + "pb"

	// Join the path parts with slashes
	packagePath := strings.Join(parts, "/")

	return fmt.Sprintf("github.com/smart-core-os/sc-bos/pkg/proto/%s", packagePath)
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
