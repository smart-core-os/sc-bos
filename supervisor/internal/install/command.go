package install

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

// outputWaitDelay is how long execRunner waits, after the command exits, for the write end of the command's
// stdout/stderr pipe to close. A child process inherits that file descriptor and can hold it open after the
// command exits, for example a daemon(8) process started by "service <name> restart".
const outputWaitDelay = 5 * time.Second

// commandRunner executes name with args and returns its combined stdout+stderr. A command that runs but
// exits non-zero yields an *exitCodeError carrying the exit code. A failed command's error includes its
// output.
type commandRunner func(ctx context.Context, name string, args ...string) (string, error)

// execRunner returns a commandRunner that runs commands with os/exec, logging each one to logger.
// It returns at most outputWaitDelay after the command exits, even if a child process still holds the write
// end of the output pipe. It then closes the read end, and discards anything written to the pipe after that.
func execRunner(logger *zap.Logger) commandRunner {
	return func(ctx context.Context, name string, args ...string) (string, error) {
		logger.Debug("exec", zap.String("cmd", name), zap.Strings("args", args))
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.WaitDelay = outputWaitDelay
		out, err := cmd.CombinedOutput()
		if errors.Is(err, exec.ErrWaitDelay) {
			// The command exited with status 0, but the output pipe was still open after outputWaitDelay.
			logger.Warn("closed command output pipe still held open by a child process",
				zap.String("cmd", name), zap.Strings("args", args))
			err = nil
		}
		if err == nil {
			return string(out), nil
		}
		wrapped := fmt.Errorf("%s %s %w", name, strings.Join(args, " "), err)
		if msg := strings.TrimSpace(string(out)); msg != "" {
			wrapped = fmt.Errorf("%w: %s", wrapped, msg)
		}
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return string(out), &exitCodeError{code: exitErr.ExitCode(), err: wrapped}
		}
		return string(out), wrapped
	}
}

type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string { return e.err.Error() }
func (e *exitCodeError) Unwrap() error { return e.err }
func (e *exitCodeError) ExitCode() int { return e.code }
