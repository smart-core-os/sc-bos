package protov1js

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
)

// Pattern to match versioned JS proto files
// e.g., smartcore/bos/meter/v1/meter_pb.js or smartcore/bos/meter/v1/meter_grpc_web_pb.js
var versionedJSPathPattern = regexp.MustCompile(`^(.+/v\d+/.+)_pb\.js$`)

// buildJSImportMapping returns a mapping from old flat import paths to new versioned paths.
// For example: "alerts_pb" -> "smartcore/bos/alerts/v1/alerts_pb".
// Only versioned _pb.js files (matching /v\d+/) are included.
func buildJSImportMapping(fsys fs.FS) (map[string]string, error) {
	mapping := make(map[string]string)

	err := fs.WalkDir(fsys, ".", func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(relPath, "_pb.js") {
			return nil
		}

		if !versionedJSPathPattern.MatchString(relPath) {
			return nil
		}

		newImport := strings.TrimSuffix(relPath, ".js")
		oldImport := path.Base(newImport)

		if oldImport != newImport {
			mapping[oldImport] = newImport
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return mapping, nil
}
