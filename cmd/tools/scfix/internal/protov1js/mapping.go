package protov1js

import (
	"fmt"
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
// A second pass adds mappings for trait-prefixed imports, e.g.:
// "traits/enter_leave_sensor_pb" -> "smartcore/bos/enterleavesensor/v1/enter_leave_sensor_pb".
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

	// Second pass: add mappings for trait-prefixed paths (e.g. "traits/enter_leave_sensor_pb").
	// These appear in node_modules under a "traits/" subdirectory and need remapping to the
	// versioned path already recorded in the first pass.
	err = fs.WalkDir(fsys, ".", func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(relPath, "_pb.js") {
			return nil
		}

		// Only process non-versioned files with a path prefix (the traits/ subdirectory).
		if versionedJSPathPattern.MatchString(relPath) {
			return nil
		}
		if !strings.Contains(relPath, "/") {
			return nil
		}

		basename := path.Base(strings.TrimSuffix(relPath, ".js")) // e.g. "enter_leave_sensor_pb"
		newImport, exists := mapping[basename]
		if !exists {
			return nil
		}

		oldImport := strings.TrimSuffix(relPath, ".js") // e.g. "traits/enter_leave_sensor_pb"
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

// buildScApiImportMapping returns a mapping from sc-api-grpc-web subpaths to sc-bos-ui-gen versioned paths.
// For example: "traits/light_pb" -> "smartcore/bos/light/v1/light_pb".
// It walks the scApiFS directory and matches basenames against the uiGenProtoFS mapping.
// Subdirectory structures (e.g. "types/time/period_pb") are handled automatically via basename matching.
func buildScApiImportMapping(uiGenProtoFS, scApiFS fs.FS) (map[string]string, error) {
	uiGenMapping, err := buildJSImportMapping(uiGenProtoFS)
	if err != nil {
		return nil, fmt.Errorf("building ui-gen import mapping: %w", err)
	}

	mapping := make(map[string]string)
	err = fs.WalkDir(scApiFS, ".", func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(relPath, "_pb.js") {
			return nil
		}

		basename := path.Base(strings.TrimSuffix(relPath, ".js")) // e.g. "light_pb"
		newImport, exists := uiGenMapping[basename]
		if !exists {
			return nil
		}

		oldSubPath := strings.TrimSuffix(relPath, ".js") // e.g. "traits/light_pb"
		mapping[oldSubPath] = newImport
		return nil
	})

	if err != nil {
		return nil, err
	}

	return mapping, nil
}
