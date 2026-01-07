package protov1

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixer"
	"github.com/smart-core-os/sc-bos/internal/compat/protopkg"
)

// protoFile represents a proto file being migrated.
type protoFile struct {
	oldPath    string // absolute path
	newPath    string // absolute path
	baseDir    string // base directory for computing import paths
	oldContent []byte
	newContent []byte
	oldPackage string   // e.g., "smartcore.bos" or "smartcore.bos.driver.dali"
	newPackage string   // e.g., "smartcore.bos.meter.v1" or "smartcore.bos.driver.dali.v1"
	types      []string // type names defined in this file
}

// getOldImportPath returns the old import path for this file.
func (f *protoFile) getOldImportPath() string {
	relPath, err := filepath.Rel(f.baseDir, f.oldPath)
	if err != nil {
		return filepath.Base(f.oldPath)
	}
	return relPath
}

// getNewImportPath returns the new import path for this file.
func (f *protoFile) getNewImportPath() string {
	relPath, err := filepath.Rel(f.baseDir, f.newPath)
	if err != nil {
		return filepath.Base(f.newPath)
	}
	return relPath
}

// collectProtoFiles discovers and prepares proto files for migration from all relevant directories.
// Files in the proto root directory are moved to new versioned structure.
// Files in other directories (internal/, pkg/) are updated in place.
func collectProtoFiles(ctx *fixer.Context) ([]protoFile, error) {
	// Directories to scan for proto files
	dirsToScan := []struct {
		path       string
		shouldMove bool
	}{
		{filepath.Join(ctx.RootDir, "proto"), true},     // Move files in proto root to versioned structure
		{filepath.Join(ctx.RootDir, "internal"), false}, // Update in place
		{filepath.Join(ctx.RootDir, "pkg"), false},      // Update in place
	}

	var allFiles []protoFile
	for _, dir := range dirsToScan {
		files, err := collectProtoFilesFromDir(ctx, dir.path, dir.shouldMove)
		if err != nil {
			// Directory might not exist, skip it
			ctx.Verbose("  Skipping directory %s: %v", dir.path, err)
			continue
		}
		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}

// collectProtoFilesFromDir discovers proto files in a specific directory.
// If shouldMove is true, files will be moved to new versioned directory structure.
// If shouldMove is false, files will be updated in place (package declaration only).
func collectProtoFilesFromDir(ctx *fixer.Context, protoDir string, shouldMove bool) ([]protoFile, error) {
	var files []protoFile
	err := filepath.WalkDir(protoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(d.Name(), ".proto") {
			return nil
		}

		filename := d.Name()
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		currentPkg := extractPackageName(content)
		if currentPkg == "" {
			ctx.Verbose("  Skipping: %s (no package)", relPath(ctx.RootDir, path))
			return nil
		}

		service, hasService := deriveServiceName(content, filename)
		newFQService := protopkg.V0ToV1(currentPkg + "." + service)
		newPackage, _ := splitPackageService(newFQService)

		// For proto files that aren't moving and without services, don't version the package.
		// This includes files like page_token.proto and other message-only files.
		if !shouldMove && !hasService {
			newPackage = currentPkg
		}

		var newPath string
		if shouldMove {
			dirPath := strings.ReplaceAll(newPackage, ".", "/")
			newPath = filepath.Join(protoDir, dirPath, filename)
		} else {
			newPath = path // Update in place
		}

		typeNames := extractTypeNames(content)

		// For files being moved, baseDir is protoDir (the proto root)
		// For files not being moved, baseDir is their containing directory (not used for imports)
		baseDir := protoDir
		if !shouldMove {
			baseDir = filepath.Dir(path)
		}

		files = append(files, protoFile{
			oldPath:    path,
			newPath:    newPath,
			baseDir:    baseDir,
			oldContent: content,
			oldPackage: currentPkg,
			newPackage: newPackage,
			types:      typeNames,
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", protoDir, err)
	}

	return files, nil
}

// writeProtoFile writes a proto file to its new location with updated content, removing the old file if needed.
// Returns true if the file has (or would be) changed, false otherwise.
func writeProtoFile(ctx *fixer.Context, file *protoFile) (bool, error) {
	if file.oldPath == file.newPath {
		// Update in place
		rel := relPath(ctx.RootDir, file.oldPath)

		if bytes.Equal(file.oldContent, file.newContent) {
			ctx.Verbose("  Unchanged: %s", rel)
			return false, nil
		}

		if ctx.DryRun {
			ctx.Info("  Would update: %s", rel)
			return true, nil
		}

		if err := os.WriteFile(file.oldPath, file.newContent, 0644); err != nil {
			return false, fmt.Errorf("writing %s: %w", file.oldPath, err)
		}

		ctx.Info("  Updated: %s", rel)
		return true, nil
	}

	// Move to new location
	oldRel := relPath(ctx.RootDir, file.oldPath)
	newRel := relPath(ctx.RootDir, file.newPath)

	if ctx.DryRun {
		ctx.Info("  Would move: %s -> %s", oldRel, newRel)
		return true, nil
	}

	targetDir := filepath.Dir(file.newPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return false, fmt.Errorf("creating directory %s: %w", targetDir, err)
	}
	if err := os.WriteFile(file.newPath, file.newContent, 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", file.newPath, err)
	}
	if err := os.Remove(file.oldPath); err != nil {
		return false, fmt.Errorf("removing %s: %w", file.oldPath, err)
	}
	ctx.Info("  Moved: %s -> %s", oldRel, newRel)
	return true, nil
}

// toTitle converts the first character of a string to uppercase.
func toTitle(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// splitPackageService splits a fully qualified name into package and service components.
func splitPackageService(fqn string) (pkg, service string) {
	lastDot := strings.LastIndex(fqn, ".")
	if lastDot == -1 {
		return "", fqn
	}
	return fqn[:lastDot], fqn[lastDot+1:]
}

// deriveServiceName extracts the service name from content, or derives it from the filename.
// Returns the service name and a boolean indicating whether a service was found in the file.
func deriveServiceName(content []byte, filename string) (string, bool) {
	service := extractFirstService(content)
	if service == "" {
		base := strings.TrimSuffix(filename, ".proto")
		service = toTitle(strings.ReplaceAll(base, "_", "")) + "Api"
		return service, false
	}
	return service, true
}

// relPath computes a relative path for logging purposes.
// Returns a path relative to rootDir, or a fallback if computation fails.
func relPath(rootDir, path string) string {
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}
