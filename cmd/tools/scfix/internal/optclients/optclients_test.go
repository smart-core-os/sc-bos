package optclients

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestOptClients(t *testing.T) {
	fixtest.Run(t, "testdata/optclients.txtar", run)
}
