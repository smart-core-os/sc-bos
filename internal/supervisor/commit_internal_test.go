package supervisor

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"go.uber.org/zap"
)

// TestRunStartupCommit_WaitsForCheckIn verifies that, when a waitForCheckIn gate is supplied, no
// commit happens until the gate releases, and then exactly one commit (plus afterCommit) runs.
func TestRunStartupCommit_WaitsForCheckIn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var commits, afters atomic.Int64
		commitFn := func(context.Context) error { commits.Add(1); return nil }
		afterCommit := func(context.Context) error { afters.Add(1); return nil }

		release := make(chan struct{})
		waitForCheckIn := func(ctx context.Context) error {
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		done := make(chan error, 1)
		go func() {
			done <- runStartupCommit(context.Background(), commitFn, "v1", waitForCheckIn, afterCommit, zap.NewNop())
		}()

		// Blocked on the gate: nothing committed yet.
		synctest.Wait()
		if got := commits.Load(); got != 0 {
			t.Fatalf("committed before check-in: %d calls", got)
		}

		close(release)
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatalf("runStartupCommit() = %v", err)
		}
		if got := commits.Load(); got != 1 {
			t.Errorf("commits = %d, want 1", got)
		}
		if got := afters.Load(); got != 1 {
			t.Errorf("afterCommit calls = %d, want 1", got)
		}
	})
}

// TestRunStartupCommit_CancelBeforeCheckIn verifies that if ctx is cancelled before the gate
// releases, no commit (and no reconcile) happens.
func TestRunStartupCommit_CancelBeforeCheckIn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var commits, afters atomic.Int64
		commitFn := func(context.Context) error { commits.Add(1); return nil }
		afterCommit := func(context.Context) error { afters.Add(1); return nil }
		waitForCheckIn := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			done <- runStartupCommit(ctx, commitFn, "v1", waitForCheckIn, afterCommit, zap.NewNop())
		}()

		synctest.Wait()
		cancel()
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatalf("runStartupCommit() = %v", err)
		}
		if got := commits.Load(); got != 0 {
			t.Errorf("commits = %d, want 0 (cancelled before check-in)", got)
		}
		if got := afters.Load(); got != 0 {
			t.Errorf("afterCommit calls = %d, want 0", got)
		}
	})
}

// TestRunStartupCommit_NoGateCommitsImmediately verifies that with no gate the commit and reconcile
// run exactly once.
func TestRunStartupCommit_NoGateCommitsImmediately(t *testing.T) {
	var commits, afters atomic.Int64
	commitFn := func(context.Context) error { commits.Add(1); return nil }
	afterCommit := func(context.Context) error { afters.Add(1); return nil }

	if err := runStartupCommit(context.Background(), commitFn, "v1", nil, afterCommit, zap.NewNop()); err != nil {
		t.Fatalf("runStartupCommit() = %v", err)
	}
	if got := commits.Load(); got != 1 {
		t.Errorf("commits = %d, want 1", got)
	}
	if got := afters.Load(); got != 1 {
		t.Errorf("afterCommit calls = %d, want 1", got)
	}
}
