package supervisor

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/supervisorpb"
)

// limits each Commit attempt, so a hung supervisor doesn't stall the retries
const commitTimeout = 10 * time.Second

// bounds the delay between failed Commit attempts
const (
	commitRetryMin = time.Second
	commitRetryMax = time.Minute
)

// CommitUntilAccepted commits version to the Supervisor, retrying with backoff until the Supervisor accepts or
// rejects the commit, or ctx is done. It returns nil once accepted, the Supervisor's error if it rejects the
// request as invalid, or ctx.Err(). The first failed attempt is logged to logger as a warning, later ones at
// debug level.
//
// The commit confirms an in-progress Supervisor update (stopping its auto-rollback), but is also safe to call
// when running a version that was previously committed.
func CommitUntilAccepted(ctx context.Context, client supervisorpb.SupervisorApiClient, version string, logger *zap.Logger) error {
	retry := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(commitRetryMin),
		backoff.WithMaxInterval(commitRetryMax),
		backoff.WithMaxElapsedTime(0),
	)
	var warned bool
	return backoff.RetryNotify(
		func() error {
			err := Commit(ctx, client, version)
			if status.Code(err) == codes.InvalidArgument {
				return backoff.Permanent(err)
			}
			return err
		},
		backoff.WithContext(retry, ctx),
		func(err error, next time.Duration) {
			log := logger.Debug
			if !warned {
				log, warned = logger.Warn, true
			}
			log("supervisor commit failed, will retry",
				zap.String("version", version), zap.Duration("retryIn", next), zap.Error(err))
		},
	)
}

// Commit makes a single attempt to commit version to the Supervisor. It waits up to 10 seconds for the
// Supervisor to accept connections, so it succeeds if the Supervisor starts within that time.
func Commit(ctx context.Context, client supervisorpb.SupervisorApiClient, version string) error {
	ctx, cancel := context.WithTimeout(ctx, commitTimeout)
	defer cancel()
	_, err := client.Commit(ctx, &supervisorpb.CommitRequest{Version: version}, grpc.WaitForReady(true))
	return err
}
