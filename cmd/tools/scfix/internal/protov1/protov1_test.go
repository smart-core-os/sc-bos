package protov1

import (
	"testing"

	"github.com/smart-core-os/sc-bos/cmd/tools/scfix/internal/fixtest"
)

// TestProtomove runs integration tests using txtar fixtures
func TestProtomove(t *testing.T) {
	fixtest.RunDir(t, "testdata", run)
}
