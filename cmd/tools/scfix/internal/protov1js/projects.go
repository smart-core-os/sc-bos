package protov1js

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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

		if !bytes.Contains(content, []byte("@smart-core-os/sc-bos-ui-gen")) {
			return nil
		}

		dirs = append(dirs, path.Dir(relPath))
		return nil
	})

	return dirs, err
}

// findNodeModules returns the path to node_modules containing @smart-core-os/sc-bos-ui-gen for projectDir.
// It walks up the directory tree until it finds node_modules with ui-gen or reaches the filesystem root.
func findNodeModules(projectDir string) string {
	current := projectDir
	for {
		nodeModules := filepath.Join(current, "node_modules")
		uiGenPath := filepath.Join(nodeModules, "@smart-core-os", "sc-bos-ui-gen", "package.json")

		if _, err := os.Stat(uiGenPath); err == nil {
			return nodeModules
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}
