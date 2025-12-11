// Package historyimports updates JavaScript/TypeScript imports to reflect the extraction
// of History aspects from main proto files into separate *_history.proto files.
//
// Example transformations:
//   - proto/transport_grpc_web_pb imports TransportHistory -> proto/transport_history_grpc_web_pb
//   - proto/transport_pb imports ListTransportHistoryRequest -> proto/transport_history_pb
//
// The package uses a strict approach for generic history imports (from history_pb/history_grpc_web_pb):
//   - Requires exactly one HistoryClient per file to determine the trait name
//   - If 0 or multiple different HistoryClients found, generic history imports are skipped
//   - Trait-specific imports (e.g., transport_pb -> transport_history_pb) always work
package historyimports

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
)

var Fix = fixer.Fix{
	ID:   "historyimports",
	Desc: "Update JS/TS imports for history aspects extracted to separate proto files",
	Run:  run,
}

func run(ctx *fixer.Context) (int, error) {
	totalChanges := 0

	// Only process UI files
	uiDir := filepath.Join(ctx.RootDir, "ui")

	err := filepath.Walk(uiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !shouldProcessUIFile(path, info) {
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

func shouldProcessUIFile(path string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	// Skip node_modules, dist, generated files
	if strings.Contains(path, "/node_modules/") ||
		strings.Contains(path, "/dist/") ||
		strings.Contains(path, "/.git/") {
		return false
	}

	// Process JS, TS, and Vue files
	ext := filepath.Ext(path)
	return ext == ".js" || ext == ".ts" || ext == ".vue"
}

func processFile(ctx *fixer.Context, filename string) (int, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	// Quick check: does file contain proto imports?
	if !bytes.Contains(content, []byte("@smart-core-os/sc-bos-ui-gen/proto/")) {
		return 0, nil
	}

	lines := strings.Split(string(content), "\n")
	newLines, changes := processLines(ctx, filename, lines)

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

// processLines processes all lines, handling both single-line and multi-line imports, and JSDoc imports.
func processLines(ctx *fixer.Context, filename string, lines []string) ([]string, int) {
	// Find the single HistoryClient in the file to determine the trait name for generic history imports.
	// If we find 0 or more than 1, we can't transform generic history imports, but we can still
	// split history symbols from trait-specific imports (e.g., transport_pb -> transport_history_pb).
	traitName, err := findHistoryClientTrait(lines)
	if err != nil && errors.Is(err, ErrMultipleHistoryClients) {
		// Multiple different HistoryClients is a hard error - skip the file entirely
		ctx.Info("! Manual import fix needed in %s: %v", relPath(ctx.RootDir, filename), err)
		return lines, 0
	}
	// For "no HistoryClient" or "no history imports", we can still process trait-specific splits
	// traitName will be empty in these cases

	var newLines []string
	var changes int
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Try single-line import first
		if processed, modified := processSingleLineImport(ctx, filename, line, traitName); modified {
			newLines = append(newLines, strings.Split(processed, "\n")...)
			changes++
			i++
			continue
		}

		// Check for multi-line import start
		if multiLineStartPattern.MatchString(line) {
			endIdx := findMultiLineImportEnd(lines, i)
			if endIdx > i {
				processed, modified := processMultiLineImport(ctx, filename, lines, i, endIdx, traitName)
				if modified {
					newLines = append(newLines, processed...)
					changes++
				} else {
					newLines = append(newLines, lines[i:endIdx+1]...)
				}
				i = endIdx + 1
				continue
			}
		}

		// Check for JSDoc import() statements
		if jsdocImportPattern.MatchString(line) {
			if traitName != "" {
				// Process JSDoc import with trait-specific replacement
				newLine, modified := processJSDocImport(line, traitName)
				if modified {
					ctx.Verbose("  Replacing JSDoc import in %s on line %d", relPath(ctx.RootDir, filename), i+1)
					changes++
					newLines = append(newLines, newLine)
					i++
					continue
				}
			}
		}

		// No match, keep line as-is
		newLines = append(newLines, line)
		i++
	}

	// If we couldn't replace JSDoc imports, warn the user
	if traitName == "" {
		detectJSDocImports(ctx, filename, newLines)
	}

	return newLines, changes
}
