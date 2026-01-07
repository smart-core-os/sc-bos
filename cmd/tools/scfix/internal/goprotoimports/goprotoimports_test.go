package goprotoimports

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestGoProtoImports(t *testing.T) {
	tests := []string{
		"testdata/single_trait.txtar",
		"testdata/multiple_traits.txtar",
		"testdata/no_gen_import.txtar",
		"testdata/package_collision.txtar",
		"testdata/driver_subdir.txtar",
		"testdata/mixed_old_new_imports.txtar",
		"testdata/comment_updates.txtar",
		"testdata/import_sorting.txtar",
		"testdata/bug_adjacent_comment.txtar",
	}

	for _, tt := range tests {
		name := strings.TrimSuffix(filepath.Base(tt), ".txtar")
		t.Run(name, func(t *testing.T) {
			fixtest.Run(t, tt, run)
		})
	}
}
