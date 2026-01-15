package wrap

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestWrapAllTraits(t *testing.T) {
	fixtest.RunDir(t, "testdata", run)
}
