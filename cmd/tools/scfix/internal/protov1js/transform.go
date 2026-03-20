package protov1js

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

// processProject returns the number of files changed in projectDir.
func processProject(ctx *fixer.Context, projectDir string) (int, error) {
	nodeModulesDir := findNodeModules(projectDir)
	if nodeModulesDir == "" {
		ctx.Verbose("  Skipping %s: no node_modules found", relPath(ctx.RootDir, projectDir))
		return 0, nil
	}

	protoDir := filepath.Join(nodeModulesDir, "@smart-core-os", "sc-bos-ui-gen", "proto")
	if _, err := os.Stat(protoDir); os.IsNotExist(err) {
		ctx.Verbose("  Skipping %s: proto directory not found in node_modules", relPath(ctx.RootDir, projectDir))
		return 0, nil
	}

	importMapping, err := buildJSImportMapping(os.DirFS(protoDir))
	if err != nil {
		return 0, fmt.Errorf("building import mapping: %w", err)
	}

	if len(importMapping) == 0 {
		ctx.Verbose("  Skipping %s: no versioned proto files found", relPath(ctx.RootDir, projectDir))
		return 0, nil
	}

	// Build cross-package mapping if sc-api-grpc-web is installed alongside sc-bos-ui-gen.
	var scApiMapping map[string]string
	if scApiDir := findScApiGrpcWeb(nodeModulesDir); scApiDir != "" {
		scApiMapping, err = buildScApiImportMapping(os.DirFS(protoDir), os.DirFS(scApiDir))
		if err != nil {
			return 0, fmt.Errorf("building sc-api import mapping: %w", err)
		}
	}

	ctx.Verbose("  Processing %s (%d versioned proto file(s))", relPath(ctx.RootDir, projectDir), len(importMapping))

	totalChanges := 0
	err = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !shouldProcessFile(path, d) {
			return nil
		}

		changes, err := processFile(ctx, path, importMapping, scApiMapping)
		if err != nil {
			return fmt.Errorf("processing %s: %w", path, err)
		}
		totalChanges += changes
		return nil
	})

	return totalChanges, err
}

// findScApiGrpcWeb returns the path to the @smart-core-os/sc-api-grpc-web package within nodeModulesDir,
// or empty string if not found.
func findScApiGrpcWeb(nodeModulesDir string) string {
	loc := filepath.Join(nodeModulesDir, "@smart-core-os", "sc-api-grpc-web")
	if _, err := os.Stat(loc); err == nil {
		return loc
	}
	return ""
}

func shouldProcessFile(path string, d os.DirEntry) bool {
	if d.IsDir() {
		return false
	}

	// Skip node_modules, dist, and git
	if strings.Contains(path, "/node_modules/") ||
		strings.Contains(path, "/dist/") ||
		strings.Contains(path, "/.git/") {
		return false
	}

	// Process JS, TS, Vue files, and TypeScript definition files
	ext := filepath.Ext(path)
	return ext == ".js" || ext == ".ts" || ext == ".vue" ||
		ext == ".jsx" || ext == ".tsx" ||
		ext == ".mjs" || ext == ".cjs"
}

func processFile(ctx *fixer.Context, filename string, importMapping map[string]string, scApiMapping map[string]string) (int, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	// Quick check: does file contain any proto imports we care about?
	hasUiGen := bytes.Contains(content, []byte("@smart-core-os/sc-bos-ui-gen/proto/"))
	hasScApi := len(scApiMapping) > 0 && bytes.Contains(content, []byte("@smart-core-os/sc-api-grpc-web/"))
	if !hasUiGen && !hasScApi {
		return 0, nil
	}

	originalContent := string(content)
	newContent := originalContent

	// Apply intrapackage transformations for sc-bos-ui-gen imports.
	for oldImport, newImport := range importMapping {
		newContent = replaceImportPaths(newContent, oldImport, newImport)
	}

	// Apply cross-package transformations from sc-api-grpc-web to sc-bos-ui-gen.
	newContent = replaceScApiImportPaths(newContent, scApiMapping)

	if newContent == originalContent {
		return 0, nil
	}

	if !ctx.DryRun {
		if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
			return 0, fmt.Errorf("writing file: %w", err)
		}
	}

	changes := countImportChanges(originalContent, newContent, importMapping) +
		countScApiImportChanges(originalContent, newContent, scApiMapping)
	ctx.Verbose("  Modified %s (%d import(s) updated)", relPath(ctx.RootDir, filename), changes)

	return changes, nil
}

// replaceImportPaths returns content with oldImport replaced by newImport in all @smart-core-os/sc-bos-ui-gen/proto/ paths.
// Handles ES6 imports, require(), dynamic import(), and JSDoc.
func replaceImportPaths(content, oldImport, newImport string) string {

	oldPath := "@smart-core-os/sc-bos-ui-gen/proto/" + oldImport
	newPath := "@smart-core-os/sc-bos-ui-gen/proto/" + newImport

	// Replace with and without .js extension
	content = strings.ReplaceAll(content, oldPath+".js", newPath+".js")
	content = strings.ReplaceAll(content, oldPath+"'", newPath+"'")
	content = strings.ReplaceAll(content, oldPath+"\"", newPath+"\"")
	content = strings.ReplaceAll(content, oldPath+")", newPath+")")

	return content
}

// replaceScApiImportPaths replaces @smart-core-os/sc-api-grpc-web/<subPath> with
// @smart-core-os/sc-bos-ui-gen/proto/<newVersionedPath> for all entries in scApiMapping.
func replaceScApiImportPaths(content string, scApiMapping map[string]string) string {
	for oldSubPath, newImport := range scApiMapping {
		oldPath := "@smart-core-os/sc-api-grpc-web/" + oldSubPath
		newPath := "@smart-core-os/sc-bos-ui-gen/proto/" + newImport

		content = strings.ReplaceAll(content, oldPath+".js", newPath+".js")
		content = strings.ReplaceAll(content, oldPath+"'", newPath+"'")
		content = strings.ReplaceAll(content, oldPath+"\"", newPath+"\"")
		content = strings.ReplaceAll(content, oldPath+")", newPath+")")
	}
	return content
}

// countScApiImportChanges returns the number of sc-api-grpc-web import paths replaced.
func countScApiImportChanges(original, updated string, scApiMapping map[string]string) int {
	count := 0
	for oldSubPath := range scApiMapping {
		oldPath := "@smart-core-os/sc-api-grpc-web/" + oldSubPath
		count += strings.Count(original, oldPath) - strings.Count(updated, oldPath)
	}
	return count
}

// countImportChanges returns the number of import paths that were changed from original to updated.
func countImportChanges(original, updated string, mapping map[string]string) int {
	count := 0
	for oldImport := range mapping {
		oldPath := "@smart-core-os/sc-bos-ui-gen/proto/" + oldImport
		originalCount := strings.Count(original, oldPath)
		updatedCount := strings.Count(updated, oldPath)
		count += originalCount - updatedCount
	}
	return count
}

func relPath(rootDir, absPath string) string {
	if rel, err := filepath.Rel(rootDir, absPath); err == nil {
		return rel
	}
	return filepath.Base(absPath)
}
