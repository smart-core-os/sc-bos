package protov1js

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestProtov1js_run(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"basic imports", "testdata/basic_imports.txtar"},
		{"grpc_web imports", "testdata/grpc_web_imports.txtar"},
		{"yarn workspace", "testdata/yarn_workspace.txtar"},
		{"no changes", "testdata/no_changes.txtar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixtest.Run(t, tt.file, run)
		})
	}
}
