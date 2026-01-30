package protov1js

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestProtov1js_run(t *testing.T) {
	fixtest.RunDir(t, "testdata", run)
}
