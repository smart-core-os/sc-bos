package supervisor

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// limits the duration of the Commit on boot, so a hung supervisor doesn't block BOS boot forever
const commitTimeout = 10 * time.Second

// RunStartupCommit commits version to the Supervisor.
//
// The commit confirms an in-progress Supervisor update (stopping its auto-rollback) or, but is also safe to call
// when running a version that was previously committed.
//
// Best effort: failure to commit is logged. If the commit fails, BOS should expect that the supervisor will roll it
// back.
func RunStartupCommit(ctx context.Context, client supervisorpb.SupervisorApiClient, version string, logger *zap.Logger) {
	ctx, cancel := context.WithTimeout(ctx, commitTimeout)
	defer cancel()
	if _, err := client.Commit(ctx, &supervisorpb.CommitRequest{Version: version}); err != nil {
		logger.Warn("supervisor commit failed", zap.String("version", version), zap.Error(err))
		return
	}
	logger.Debug("supervisor commit succeeded", zap.String("version", version))
}
