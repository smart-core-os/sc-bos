package gentrait

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestGenProtoMv(t *testing.T) {
	fixtest.RunDir(t, "testdata/move", runMove)
}
