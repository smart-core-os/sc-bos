package gentrait

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestGenProtoRef(t *testing.T) {
	fixtest.RunDir(t, "testdata/refs", runUpdateRefs)
}
