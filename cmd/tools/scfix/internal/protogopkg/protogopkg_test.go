package protogopkg

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestProtoGoPkg(t *testing.T) {
	tests := []string{
		"testdata/comprehensive.txtar",
		"testdata/no_change.txtar",
	}

	for _, tt := range tests {
		name := strings.TrimSuffix(filepath.Base(tt), ".txtar")
		t.Run(name, func(t *testing.T) {
			fixtest.Run(t, tt, run)
		})
	}
}
