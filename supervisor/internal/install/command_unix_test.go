//go:build unix

package install

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// A command that exits non-zero yields an *exitCodeError whose message includes the command's output.
func TestExecRunner_ExitError(t *testing.T) {
	run := execRunner(zap.NewNop())
	// The output is built from $0, so it doesn't appear in the command's arguments.
	_, err := run(context.Background(), "sh", "-c", `echo "$0 does not exist" >&2; exit 3`, "sc_bos")
	codeErr, ok := errors.AsType[*exitCodeError](err)
	if !ok || codeErr.ExitCode() != 3 {
		t.Fatalf("error = %v, want *exitCodeError with code 3", err)
	}
	if !strings.Contains(err.Error(), "sc_bos does not exist") {
		t.Errorf("error = %q, want it to include the command output", err)
	}
}
