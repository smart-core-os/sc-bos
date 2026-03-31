package gentrait

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/go/gopkg"
)

// specialCaseMappings maps gentrait directory names to their proto directory paths
// when they don't follow the standard naming convention.
// Unfortunately there's no foolproof way to determine this at runtime from the information in the gentrait pkg,
// and no way to statically generate this mapping.
// All these are effectively typos in the gentrait package.
var specialCaseMappings = map[string]string{
	"dalipb":    "driver/dalipb",
	"lighttest": "lightingtestpb",
	"servicepb": "servicespb",
}

// gentraitToProtoFilePath converts a relative gentrait file path to an absolute proto file path.
// Examples:
//   - meter/info.go -> {rootDir}/pkg/proto/meterpb/info.go
//   - wastepb/model.go -> {rootDir}/pkg/proto/wastepb/model.go
//   - healthpb/standard/common.go -> {rootDir}/pkg/proto/healthpb/standard/common.go
func gentraitToProtoFilePath(rootDir, relPath string) (string, error) {
	relPath = filepath.ToSlash(relPath)
	dir, filename := path.Split(relPath)

	if dir == "." || filename == relPath {
		return "", fmt.Errorf("path too short: %s", relPath)
	}

	protoPath := gentraitToProtoImportPath(dir)
	destPath := filepath.Join(rootDir, "pkg", "proto", filepath.FromSlash(protoPath), filename)
	return destPath, nil
}

// relPathToProtoImport returns both the package name and full import path for a file's new location.
// For example, "healthpb/model.go" returns ("healthpb", "github.com/smart-core-os/sc-bos/pkg/proto/healthpb")
// For example, "healthpb/utils/util.go" returns ("utils", "github.com/smart-core-os/sc-bos/pkg/proto/healthpb/utils")
func relPathToProtoImport(relPath string) (packageName string, fullImportPath string) {
	dir := path.Dir(filepath.ToSlash(relPath))

	if dir == "." {
		return "", "github.com/smart-core-os/sc-bos/pkg/proto/"
	}

	// Construct full import path
	protoPath := gentraitToProtoImportPath(dir)
	fullImportPath = "github.com/smart-core-os/sc-bos/pkg/proto/" + protoPath

	// Derive package name from the import path using the standard logic
	packageName = gopkg.ImportPathToAssumedName(fullImportPath)

	return packageName, fullImportPath
}

// gentraitToProtoImportPath converts a gentrait import path to proto import path.
// Adds 'pb' suffix to the first component if needed, preserving subdirectories.
// Examples:
//   - meter -> meterpb
//   - wastepb -> wastepb
//   - healthpb/utils -> healthpb/utils
//   - dalipb -> driver/dalipb (special case)
//   - lighttest -> lightingtestpb (special case)
func gentraitToProtoImportPath(gentraitPath string) string {
	parts := strings.Split(gentraitPath, "/")
	firstDir := parts[0]

	// Check for special case mappings first
	if mappedPath, ok := specialCaseMappings[firstDir]; ok {
		// Replace the first component with the mapped path
		if len(parts) > 1 {
			// Preserve subdirectories
			return path.Join(append([]string{mappedPath}, parts[1:]...)...)
		}
		return mappedPath
	}

	// Add 'pb' suffix to first component if needed
	if !strings.HasSuffix(parts[0], "pb") {
		parts[0] += "pb"
	}

	return path.Join(parts...)
}
