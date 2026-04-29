package scgolang

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

func TestScgolangmv(t *testing.T) {
	fixtest.RunDir(t, "testdata", run)
}
