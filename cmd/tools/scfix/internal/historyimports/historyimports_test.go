package historyimports

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestHistoryImports(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"mixed imports", "testdata/mixed_imports.txtar"},
		{"multiple traits", "testdata/multiple_traits.txtar"},
		{"vue file", "testdata/vue_file.txtar"},
		{"with .js extension", "testdata/with_js_extension.txtar"},
		{"no history symbols", "testdata/no_history.txtar"},
		{"history proto", "testdata/history_proto.txtar"},
		{"no history client", "testdata/no_history_client.txtar"},
		{"multiple history clients", "testdata/multiple_history_clients.txtar"},
		{"jsdoc imports", "testdata/jsdoc_imports.txtar"},
		{"jsdoc no replacement", "testdata/jsdoc_no_replacement.txtar"},
		{"jsdoc non history", "testdata/jsdoc_non_history.txtar"},
		{"already correct", "testdata/already_correct.txtar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixtest.Run(t, tt.file, run)
		})
	}
}
