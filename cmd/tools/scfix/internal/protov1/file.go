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

const scBOSModule = "github.com/smart-core-os/sc-bos"

// protoFile represents a proto file being migrated.
type protoFile struct {
	oldPath        string   // absolute path
	newPath        string   // absolute path
	baseDir        string   // base directory for computing import paths
	altImportPaths []string // additional import path keys (e.g. sc-api-relative paths)
	oldContent     []byte
	newContent     []byte
	oldPackage     string            // e.g., "smartcore.bos" or "smartcore.bos.driver.dali"
	newPackage     string            // e.g., "smartcore.bos.meter.v1" or "smartcore.bos.driver.dali.v1"
	newGoPackage   string            // if non-empty, update option go_package to this value
	serviceRenames map[string]string // maps old service names to new service names
	types          []string          // type names defined in this file
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
	protoRoot := filepath.Join(ctx.RootDir, "proto")

	// Directories to scan for proto files.
	// destDir, if non-empty, is the base for computing new file paths when shouldMove is true.
	// importBase, if non-empty, is used to compute alt import paths (how files in this dir
	// are referenced in import statements by other protos using protoc's include path).
	scApiProtobuf := filepath.Join(ctx.RootDir, "sc-api", "protobuf")
	dirsToScan := []struct {
		path       string
		shouldMove bool
		destDir    string // overrides scan dir as the base for moved files
		importBase string // base for computing alt import paths (e.g. sc-api/protobuf/)
	}{
		{protoRoot, true, "", ""},                                                                    // Move files in proto root to versioned structure
		{filepath.Join(ctx.RootDir, "internal"), false, "", ""},                                      // Update in place
		{filepath.Join(ctx.RootDir, "pkg"), false, "", ""},                                           // Update in place
		{filepath.Join(ctx.RootDir, "sc-api", "protobuf", "traits"), true, protoRoot, scApiProtobuf}, // sc-api traits -> proto/
		{filepath.Join(ctx.RootDir, "sc-api", "protobuf", "info"), true, protoRoot, scApiProtobuf},   // sc-api info -> proto/
		{filepath.Join(ctx.RootDir, "sc-api", "protobuf", "types"), true, protoRoot, scApiProtobuf},  // sc-api types -> proto/
	}

	var allFiles []protoFile
	for _, dir := range dirsToScan {
		files, err := collectProtoFilesFromDir(ctx, dir.path, dir.shouldMove, dir.destDir, dir.importBase)
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
// destDir, if non-empty, overrides protoDir as the base directory for moved files (used for sc-api files).
// importBase, if non-empty, is used to compute alt import paths recording how files in this dir
// are referenced in import statements by other protos (e.g. "types/unit.proto" relative to sc-api/protobuf/).
func collectProtoFilesFromDir(ctx *fixer.Context, protoDir string, shouldMove bool, destDir string, importBase string) ([]protoFile, error) {
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

		// Choose the FQN conversion function based on the source package.
		var convertFQN func(string) string
		switch {
		case currentPkg == "smartcore.traits":
			convertFQN = protopkg.TraitsToV1
		case currentPkg == "smartcore.info":
			convertFQN = protopkg.InfoToV1
		case strings.HasPrefix(currentPkg, "smartcore.types"):
			convertFQN = protopkg.TypesToV1
		default:
			convertFQN = protopkg.V0ToV1
		}

		// Extract all services and build rename mapping
		services := extractAllServices(content)
		serviceRenames := make(map[string]string)

		// For determining newPackage, use the first service (or derive from filename if no service)
		var newPackage string
		if len(services) > 0 {
			// Use first service to determine the new package
			firstService := services[0]
			newFQService := convertFQN(currentPkg + "." + firstService)
			newPackage, _ = splitPackageService(newFQService)

			// Build rename map for all services and validate they all map to the same package
			for _, service := range services {
				oldFQN := currentPkg + "." + service
				newFQN := convertFQN(oldFQN)
				servicePkg, newService := splitPackageService(newFQN)

				// Check if this service maps to a different package than the first service
				if servicePkg != newPackage {
					return fmt.Errorf(
						"file %s contains multiple services that map to different packages: %s -> %s, %s -> %s",
						relPath(ctx.RootDir, path),
						firstService, newPackage,
						service, servicePkg,
					)
				}

				if service != newService {
					serviceRenames[service] = newService
				}
			}
		} else {
			// No services in file, use filename to determine package
			derivedService := serviceNameFromFileName(filename)
			newFQService := convertFQN(currentPkg + "." + derivedService)
			newPackage, _ = splitPackageService(newFQService)
		}

		// For proto files that aren't moving and without services, don't version the package.
		// This includes files like page_token.proto and other message-only files.
		if !shouldMove && len(services) == 0 {
			newPackage = currentPkg
		}

		// dest is the base directory for moved files.
		dest := destDir
		if dest == "" {
			dest = protoDir
		}

		var newPath string
		if shouldMove {
			dirPath := strings.ReplaceAll(newPackage, ".", "/")
			newPath = filepath.Join(dest, dirPath, filename)
		} else {
			newPath = path // Update in place
		}

		typeNames := extractTypeNames(content)

		// For files being moved, baseDir is dest (the proto root).
		// For files not being moved, baseDir is their containing directory (not used for imports).
		baseDir := dest
		if !shouldMove {
			baseDir = filepath.Dir(path)
		}

		// Compute updated go_package for files being migrated or corrected.
		var newGoPackage string
		if shouldMove {
			switch {
			case currentPkg == "smartcore.traits":
				// newPackage is like "smartcore.bos.meter.v1" — strip ".v1" to get resource segment.
				pkgWithoutV1 := strings.TrimSuffix(newPackage, ".v1")
				resource := pkgWithoutV1[strings.LastIndex(pkgWithoutV1, ".")+1:]
				newGoPackage = scBOSModule + "/pkg/proto/" + resource + "pb"
			case currentPkg == "smartcore.info":
				newGoPackage = scBOSModule + "/pkg/proto/infopb"
			case currentPkg == "smartcore.types":
				newGoPackage = scBOSModule + "/pkg/proto/typespb"
			case strings.HasPrefix(currentPkg, "smartcore.types."):
				// Subpackages of types get a distinct go package to avoid filename collisions
				// (e.g. both types/ and types/time/ have unit.proto).
				sub := strings.TrimPrefix(currentPkg, "smartcore.types.")
				newGoPackage = scBOSModule + "/pkg/proto/" + sub + "pb"
			}
		}

		// Compute alt import paths for sc-api files: the path relative to importBase
		// (e.g. "types/unit.proto" or "types/time/unit.proto" relative to sc-api/protobuf/).
		// This allows updateImportPaths to prefer exact path matches over basename fallbacks,
		// avoiding collisions where multiple files share the same basename (e.g. unit.proto).
		var altImportPaths []string
		if importBase != "" {
			if rel, err := filepath.Rel(importBase, path); err == nil {
				altImportPaths = []string{rel}
			}
		}
		// For already-migrated sc-api types/info files (already in proto/ from a prior fixer run),
		// add their old sc-api-relative import paths as alt paths so that other files still
		// using old import strings (e.g. "types/unit.proto") can resolve them correctly.
		if importBase == "" {
			switch currentPkg {
			case "smartcore.bos.types.v1":
				altImportPaths = append(altImportPaths, "types/"+filename)
			case "smartcore.bos.info.v1":
				altImportPaths = append(altImportPaths, "info/"+filename)
			}
			// Handle arbitrary types subpackages (e.g. smartcore.bos.types.time.v1)
			if strings.HasPrefix(currentPkg, "smartcore.bos.types.") &&
				strings.HasSuffix(currentPkg, ".v1") &&
				currentPkg != "smartcore.bos.types.v1" {
				// e.g. "smartcore.bos.types.time.v1" → sub = "time"
				middle := strings.TrimSuffix(strings.TrimPrefix(currentPkg, "smartcore.bos.types."), ".v1")
				altImportPaths = append(altImportPaths, "types/"+middle+"/"+filename)
			}
		}

		pf := protoFile{
			oldPath:        path,
			newPath:        newPath,
			baseDir:        baseDir,
			altImportPaths: altImportPaths,
			oldContent:     content,
			oldPackage:     currentPkg,
			newPackage:     newPackage,
			newGoPackage:   newGoPackage,
			serviceRenames: serviceRenames,
			types:          typeNames,
		}

		// For sc-api traits files being moved, check for conflicts in the destination.
		if currentPkg == "smartcore.traits" && shouldMove {
			conflict, err := checkTraitsConflict(ctx, pf)
			if err != nil {
				return fmt.Errorf("checking conflicts for %s: %w", relPath(ctx.RootDir, path), err)
			}
			if conflict != "" {
				ctx.Info("  Skipping sc-api trait (conflict): %s — %s", relPath(ctx.RootDir, path), conflict)
				return nil
			}
		}

		files = append(files, pf)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", protoDir, err)
	}

	return files, nil
}

// checkTraitsConflict checks whether moving pf to its target directory would conflict
// with proto files already present there. Returns a non-empty conflict description when
// there is an overlap in service or type names; returns "" when the move is safe.
func checkTraitsConflict(ctx *fixer.Context, pf protoFile) (string, error) {
	targetDir := filepath.Dir(pf.newPath)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return "", nil // Case 1: target dir doesn't exist yet — safe to move
	}

	// Read existing proto files in target dir and collect their services and types.
	existingServices := make(map[string]bool)
	existingTypes := make(map[string]bool)

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", targetDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".proto") {
			continue
		}
		existing, err := os.ReadFile(filepath.Join(targetDir, e.Name()))
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		for _, svc := range extractAllServices(existing) {
			existingServices[svc] = true
		}
		for _, t := range extractTypeNames(existing) {
			existingTypes[t] = true
		}
	}

	// Check incoming services/types against existing ones.
	incomingServices := extractAllServices(pf.oldContent)
	for _, svc := range incomingServices {
		// Use the potentially-renamed service name for the conflict check.
		name := svc
		if renamed, ok := pf.serviceRenames[svc]; ok {
			name = renamed
		}
		if existingServices[name] {
			return fmt.Sprintf("service %s already exists in %s", name, relPath(ctx.RootDir, targetDir)), nil
		}
	}
	incomingTypes := extractTypeNames(pf.oldContent)
	for _, t := range incomingTypes {
		if existingTypes[t] {
			return fmt.Sprintf("type %s already exists in %s", t, relPath(ctx.RootDir, targetDir)), nil
		}
	}

	return "", nil // Case 2: target dir exists but no overlap — safe to merge
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

// serviceNameFromFileName derives a service name from a proto filename.
// This is used to determine the package for files without services (message-only files).
func serviceNameFromFileName(filename string) string {
	base := strings.TrimSuffix(filename, ".proto")
	return toTitle(strings.ReplaceAll(base, "_", "")) + "Api"
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
