// Package protofile provides utilities for discovering and parsing protocol buffer files.
package protofile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Discover finds all .proto files in protoDir.
// Returns paths relative to protoDir.
func Discover(protoDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(protoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".proto") {
			return nil
		}

		relPath, err := filepath.Rel(protoDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}

		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
