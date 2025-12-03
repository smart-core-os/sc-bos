package wrap

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestWrapAllTraits(t *testing.T) {
	fixtest.Run(t, "testdata/wrap_all_traits.txtar", run)
}
