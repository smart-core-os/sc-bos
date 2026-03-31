package nodeclient

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestFix(t *testing.T) {
	fixtest.RunDir(t, "testdata", run)
}
