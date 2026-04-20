package cloud

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/util/concurrent"
)

// AutoPoll runs the deployment polling loop until ctx is cancelled or a new deployment
// requires a reboot. It watches conn for registration changes and resets the poll
// timer to fire immediately whenever the registration changes.
// Returns true when a reboot is required.
//
// At start, AutoPoll waits a random duration in [0, min(interval, 1 minute)) before its first poll,
// to spread check-in load across many nodes starting simultaneously.
func AutoPoll(ctx context.Context, conn *Conn, interval time.Duration, logger *zap.Logger) bool {
	initial, changes := conn.PullState(ctx)
	changes = concurrent.BreakBackpressure(changes) // only care about the latest value, drop others
	cur := initial.Registration

	var initialTick <-chan time.Time
	if cur != nil {
		initialInterval := min(interval, time.Minute)
		ns := rand.Int64N(initialInterval.Nanoseconds())
		initialTick = time.After(time.Duration(ns))
	}
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return false
		case snap, ok := <-changes:
			if !ok {
				return false
			}
			if sameRegistration(cur, snap.Registration) {
				continue // health-only update, not a registration change
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
		}

		if cur == nil {
			continue
		}
		logger.Debug("checking for deployment updates",
			zap.String("bosapi", cur.BosapiRoot),
			zap.String("clientId", cur.ClientID))
		needReboot, err := conn.Update(ctx)
		if errors.Is(err, ErrNotRegistered) {
			continue
		} else if err != nil {
			logger.Error("failed to check for deployment updates", zap.Error(err))
		}
		if needReboot {
			return true
		}
	}
}

func sameRegistration(a, b *Registration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
