package protogopkg

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestProtoGoPkg(t *testing.T) {
	fixtest.RunDir(t, "testdata", run)
}
