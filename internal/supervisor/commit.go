package supervisor

import (
	"context"

	"go.uber.org/zap"
)

// RunStartupCommit commits version to the Supervisor exactly once, then runs afterCommit.
//
// If waitForCheckIn is non-nil it blocks on it first (until the first successful cloud check-in), so a
// cloud-managed node confirms its running version to the Supervisor only after proving it can reach
// SCC. If waitForCheckIn returns an error (ctx cancelled before the first check-in) no commit is made,
// leaving any in-flight update for the Supervisor's own deadline to roll back.
//
// The commit confirms an in-progress Supervisor update (stopping its auto-rollback) or, when none is in
// flight, records the running version as the rollback baseline. afterCommit, when non-nil, runs once
// after the commit to reconcile any in-flight update outcome with SCC.
//
// RunStartupCommit is intended to be launched as its own goroutine (or errgroup.Go entry). Every step
// is best-effort: failures are logged, and it always returns nil so it never fails BOS startup.
func RunStartupCommit(ctx context.Context, client *Client, version string, waitForCheckIn, afterCommit func(context.Context) error, logger *zap.Logger) error {
	commitFn := func(ctx context.Context) error {
		return client.Commit(ctx, version)
	}
	return runStartupCommit(ctx, commitFn, version, waitForCheckIn, afterCommit, logger)
}

// runStartupCommit is the testable core of RunStartupCommit: it takes a plain commitFn so tests can
// drive it without pulling gRPC network goroutines into a testing/synctest bubble.
func runStartupCommit(ctx context.Context, commitFn func(context.Context) error, version string, waitForCheckIn, afterCommit func(context.Context) error, logger *zap.Logger) error {
	if waitForCheckIn != nil {
		if err := waitForCheckIn(ctx); err != nil {
			// ctx cancelled before the first successful check-in; do not commit.
			logger.Debug("supervisor commit skipped: no successful check-in before shutdown", zap.Error(err))
			return nil
		}
	}

	if err := commitFn(ctx); err != nil {
		logger.Warn("supervisor commit failed", zap.String("version", version), zap.Error(err))
	} else {
		logger.Debug("supervisor commit succeeded", zap.String("version", version))
	}

	// After the commit the Supervisor has confirmed/settled this version, so reconcile any in-flight
	// update outcome with SCC. Best-effort: a failure self-corrects on the next poll.
	if afterCommit != nil {
		if err := afterCommit(ctx); err != nil {
			logger.Warn("post-commit update reconciliation failed", zap.Error(err))
		}
	}
	return nil
}
