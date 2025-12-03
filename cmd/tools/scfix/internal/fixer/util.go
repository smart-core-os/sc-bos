package fixer

import (
	"os"
	"strings"
)

// ShouldProcessFile determines whether a file should be processed by transforms.
func ShouldProcessFile(path string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}

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
