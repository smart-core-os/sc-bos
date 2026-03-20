package protov1js

import (
	"bytes"
	"io/fs"
	"path"
	"strings"
)

// findProjectDirs returns relative paths to directories containing package.json files
// that depend on @smart-core-os/sc-bos-ui-gen.
func findProjectDirs(fsys fs.FS) ([]string, error) {
	var dirs []string

	err := fs.WalkDir(fsys, ".", func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || (strings.HasPrefix(name, ".") && relPath != ".") {
				return fs.SkipDir
			}
			return nil
		}

		if d.Name() != "package.json" {
			return nil
		}

		content, err := fs.ReadFile(fsys, relPath)
		if err != nil {
			return nil
		}

		if !bytes.Contains(content, []byte("@smart-core-os/sc-bos-ui-gen")) &&
			!bytes.Contains(content, []byte("@smart-core-os/sc-api-grpc-web")) {
			return nil
		}

		dirs = append(dirs, path.Dir(relPath))
		return nil
	})

	return dirs, err
}
