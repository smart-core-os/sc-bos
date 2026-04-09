package gentrait

import (
	"go/ast"
	"strings"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/go/gopkg"
)

const (
	gentraitImportPrefix = "github.com/smart-core-os/sc-bos/pkg/gentrait/"
	protoImportPrefix    = "github.com/smart-core-os/sc-bos/pkg/proto/"
)

// getImportPackageName returns the package name for an import spec.
// If the import has an alias, returns the alias. Otherwise derives the assumed name from the path.
func getImportPackageName(impSpec *ast.ImportSpec) string {
	if impSpec.Name != nil {
		return impSpec.Name.Name
	}
	importPath := strings.Trim(impSpec.Path.Value, `"`)
	return gopkg.ImportPathToAssumedName(importPath)
}

// isGentraitImport checks if the import path is from pkg/gentrait.
func isGentraitImport(importPath string) bool {
	return strings.HasPrefix(importPath, gentraitImportPrefix)
}

// isProtoImport checks if the import path is from pkg/proto.
func isProtoImport(importPath string) bool {
	return strings.HasPrefix(importPath, protoImportPrefix)
}

// convertGentraitImportToProto converts a gentrait import path to its proto equivalent.
// Returns the new proto import path.
func convertGentraitImportToProto(importPath string) string {
	gentraitPath := strings.TrimPrefix(importPath, gentraitImportPrefix)
	protoPath := gentraitToProtoImportPath(gentraitPath)
	return protoImportPrefix + protoPath
}
