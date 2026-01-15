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

	ctx.Verbose("  Processing %s (%d versioned proto file(s))", relPath(ctx.RootDir, projectDir), len(importMapping))

	totalChanges := 0
	err = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !shouldProcessFile(path, d) {
			return nil
		}

		changes, err := processFile(ctx, path, importMapping)
		if err != nil {
			return fmt.Errorf("processing %s: %w", path, err)
		}
		totalChanges += changes
		return nil
	})

	return totalChanges, err
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

func processFile(ctx *fixer.Context, filename string, importMapping map[string]string) (int, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	// Quick check: does file contain proto imports?
	if !bytes.Contains(content, []byte("@smart-core-os/sc-bos-ui-gen/proto/")) {
		return 0, nil
	}

	originalContent := string(content)
	newContent := originalContent

	// Apply all transformations
	for oldImport, newImport := range importMapping {
		newContent = replaceImportPaths(newContent, oldImport, newImport)
	}

	if newContent == originalContent {
		return 0, nil
	}

	if !ctx.DryRun {
		if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
			return 0, fmt.Errorf("writing file: %w", err)
		}
	}

	changes := countImportChanges(originalContent, newContent, importMapping)
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
