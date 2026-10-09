package install

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var errCommand = errors.New("command failed")

// step is one command the installer is expected to run, paired with the result the fake returns for it.
type step struct {
	want []string // the command expected next, name followed by args
	out  string   // output returned to the installer
	err  error    // error returned to the installer
}

// expect builds a step that expects cmd and returns out with no error.
func expect(out string, cmd ...string) step { return step{want: cmd, out: out} }

// expectErr builds a step that expects cmd and returns a generic (non-exit-coded) error.
func expectErr(cmd ...string) step { return step{want: cmd, err: errCommand} }

// expectExit builds a step that expects cmd and returns an *exitCodeError with the given exit code, the
// way execRunner surfaces a command that ran but exited non-zero (e.g. podman image exists). This lets
// tests drive exit-code handling without the real command.
func expectExit(code int, cmd ...string) step {
	return step{want: cmd, err: &exitCodeError{code: code, err: errCommand}}
}

// fakeRunner replays steps in order: each command must match the next expected one or the test aborts
// immediately, and the step's prepared result is returned.
type fakeRunner struct {
	t     *testing.T
	steps []step
	next  int
}

// newFakeRunner returns a commandRunner that replays steps, and fails the test at cleanup if any step
// was not run.
func newFakeRunner(t *testing.T, steps ...step) commandRunner {
	t.Helper()
	f := &fakeRunner{t: t, steps: steps}
	t.Cleanup(func() {
		if !t.Failed() && f.next != len(f.steps) {
			t.Errorf("ran %d commands, want %d", f.next, len(f.steps))
		}
	})
	return f.run
}

func (f *fakeRunner) run(_ context.Context, name string, args ...string) (string, error) {
	f.t.Helper()
	got := append([]string{name}, args...)
	if f.next >= len(f.steps) {
		f.t.Fatalf("unexpected command: %s", strings.Join(got, " "))
	}
	s := f.steps[f.next]
	if diff := cmp.Diff(s.want, got); diff != "" {
		f.t.Fatalf("command %d mismatch (-want +got):\n%s", f.next, diff)
	}
	f.next++
	return s.out, s.err
}
