//go:build unix

package install

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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

// execRunner returns soon after the command exits, even while a child process holds the write end of the
// command's stdout pipe. Here sh starts "sleep 60" in the background, so sleep inherits sh's stdout. sh then
// prints sleep's PID and exits. execRunner returns that output and no error, without waiting for sleep to exit.
func TestExecRunner_ChildHoldsOutputPipe(t *testing.T) {
	run := execRunner(zap.NewNop())
	start := time.Now()
	out, err := run(context.Background(), "sh", "-c", `sleep 60 & echo $!`)
	elapsed := time.Since(start)
	pid, convErr := strconv.Atoi(strings.TrimSpace(out))
	if convErr == nil {
		t.Cleanup(func() {
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Kill()
			}
		})
	}
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if convErr != nil {
		t.Errorf("output = %q, want the child's pid", out)
	}
	if elapsed > 30*time.Second {
		t.Errorf("run() took %s, want it not to wait for the child", elapsed)
	}
}
