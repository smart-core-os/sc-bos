package install

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// commandRunner executes name with args and returns its combined stdout+stderr. A command that runs but
// exits non-zero yields an *exitCodeError carrying the exit code. A failed command's error includes its
// output.
type commandRunner func(ctx context.Context, name string, args ...string) (string, error)

// execRunner returns a commandRunner that runs commands with os/exec, logging each one to logger.
func execRunner(logger *zap.Logger) commandRunner {
	return func(ctx context.Context, name string, args ...string) (string, error) {
		logger.Debug("exec", zap.String("cmd", name), zap.Strings("args", args))
		out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
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
