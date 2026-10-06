package cloud

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/util/concurrent"
)

// checkInRetryMin is the initial delay before retrying a failed check-in.
const checkInRetryMin = 5 * time.Second

// AutoPoll runs the deployment polling loop until ctx is cancelled or a new deployment
// requires a reboot. It watches conn for registration changes and resets the poll
// timer to fire immediately whenever the registration changes.
// Returns true when a reboot is required.
//
// Startup behavior depends on whether the supervisor reports it is performing an update. If so,
// then we are probably running a newly updated version, and should Commit this immediately (before the
// supervisor's deadline). Therefore, there is no delay before the first poll.
// Otherwise, AutoPoll waits a random duration in [0, min(interval, 1 minute)) before its first poll,
// to spread check-in load across many nodes starting simultaneously.
//
// A check-in that can't connect to the cloud is retried with exponential backoff, from checkInRetryMin up to the
// poll interval, so the node shows as online in the cloud as soon as possible. This stops after the first check-in
// that connects (even if a following step fails, such as a failed install or an unavailable Supervisor. From then
// on, failures wait for the next poll.
func AutoPoll(ctx context.Context, conn *Conn, interval time.Duration, logger *zap.Logger) bool {
	initial, changes := conn.PullState(ctx)
	changes = concurrent.BreakBackpressure(changes) // only care about the latest value, drop others
	cur := initial.Registration

	var initialTick <-chan time.Time
	if cur != nil {
		initialTick = time.After(initialPollDelay(ctx, conn, interval, logger))
	}
	ticker := time.NewTicker(interval)
	var (
		reachedCloud bool // a check-in has connected to the cloud, or failed for another reason, since AutoPoll started
		retryTick    <-chan time.Time
	)
	retry := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(checkInRetryMin),
		backoff.WithMaxInterval(interval),
		backoff.WithMaxElapsedTime(0),
	)
	for {
		select {
		case <-ctx.Done():
			return false
		case snap, ok := <-changes:
			if !ok {
				return false
			}
			if sameRegistration(cur, snap.Registration) {
				continue // health-only update (or renewal), not a (re-)enrollment change
			}
			cur = snap.Registration
			if cur == nil {
				continue // keep waiting
			}
			// reset timer phase
			ticker.Reset(interval)
			// update immediately
		case <-ticker.C:
		case <-initialTick:
			// reset ticker phase
			ticker.Reset(interval)
		case <-retryTick:
			// reset ticker phase
			ticker.Reset(interval)
		}
		retryTick = nil

		if cur == nil {
			continue
		}
		logger.Debug("checking for deployment updates",
			zap.String("nodeId", cur.NodeID()))
		needReboot, err := conn.Update(ctx)
		if errors.Is(err, ErrNotRegistered) {
			continue
		} else if err != nil {
			logger.Error("failed to check for deployment updates", zap.Error(err))
		}
		if _, installFailed := errors.AsType[installError](err); !IsConnectionError(err) || installFailed {
			reachedCloud = true
		} else if !reachedCloud {
			retryTick = time.After(retry.NextBackOff())
		}
		if needReboot {
			return true
		}
	}
}

// initialPollDelay returns how long AutoPoll waits before its first poll: a random duration in
// [0, min(interval, 1 minute)), or zero when the Supervisor is downloading or installing an update.
func initialPollDelay(ctx context.Context, conn *Conn, interval time.Duration, logger *zap.Logger) time.Duration {
	inFlight, err := conn.supervisorBusy(ctx)
	if err != nil {
		logger.Debug("failed to get supervisor update status, delaying first check-in", zap.Error(err))
	}
	if inFlight {
		logger.Debug("supervisor update in progress, checking in now")
		return 0
	}
	initialInterval := min(interval, time.Minute)
	return time.Duration(rand.Int64N(initialInterval.Nanoseconds()))
}

// sameRegistration reports whether a and b represent the same enrolled identity.
// It compares node id (stable across renewals), so a routine certificate renewal
// is not treated as a re-enrollment that would reset the poll phase.
func sameRegistration(a, b *Registration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.NodeID() == b.NodeID()
}
