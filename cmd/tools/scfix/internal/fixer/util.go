package fixer

import (
	"io/fs"
	"os"
	"strings"
)

// ShouldProcessFile determines whether a file should be processed by transforms.
func ShouldProcessFile(path string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	return shouldProcessPath(path)
}

// ShouldProcessFileDir determines whether a file should be processed by transforms.
func ShouldProcessFileDir(path string, info fs.DirEntry) bool {
	if info.IsDir() {
		return false
	}
	return shouldProcessPath(path)
}

func shouldProcessPath(path string) bool {
	if !strings.HasSuffix(path, ".go") {
		return false
	}

	if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
		return false
	}

	if strings.Contains(path, "/cmd/tools/scfix/internal/") && strings.Contains(path, "/testdata/") {
		return false
	}

	return true
}
