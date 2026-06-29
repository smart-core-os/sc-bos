package cloud

import (
	"context"
	"crypto/x509"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/util/concurrent"
)

// renewalProgress is the fraction of a certificate's lifetime at which renewal
// begins (the contract's "roughly two-thirds of its lifetime").
const renewalProgress = 2.0 / 3.0

// renewalRetryInterval is how long to wait before retrying a failed renewal
// while the certificate is still valid.
const renewalRetryInterval = time.Hour

// AutoRenew renews the controller's certificate before it expires, hot-swapping
// the new certificate with no restart. It begins attempting renewal at roughly
// two-thirds of the certificate's lifetime and keeps retrying while online and
// in-window. It does NOT retry once the certificate has expired — an expired
// certificate cannot authenticate the renewal endpoint, so recovery is
// re-enrollment with a fresh code (a credential change re-arms the loop).
//
// It runs until ctx is cancelled.
func AutoRenew(ctx context.Context, conn *Conn, logger *zap.Logger) {
	initial, changes := conn.PullState(ctx)
	changes = concurrent.BreakBackpressure(changes)
	cur := initial.Registration

	timer := time.NewTimer(time.Hour)
	timer.Stop()
	arm := func(reg *Registration) {
		timer.Stop()
		if reg == nil {
			return
		}
		d := untilRenew(reg.Leaf(), time.Now())
		if d < 0 {
			logger.Warn("cloud certificate has expired; re-enrollment required",
				zap.String("nodeId", reg.NodeID()))
			return
		}
		timer.Reset(d)
	}
	arm(cur)

	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-changes:
			if !ok {
				return
			}
			if sameRegistration(cur, snap.Registration) {
				// Same identity (node id) — but a renewal swaps the certificate, so
				// re-arm to the new expiry whenever the leaf actually changed. Compare
				// the certificate itself, not just NotAfter: a renewed cert always has
				// a fresh serial even if the validity window happens to coincide.
				if cur != nil && snap.Registration != nil &&
					!cur.Leaf().Equal(snap.Registration.Leaf()) {
					cur = snap.Registration
					arm(cur)
				}
				continue
			}
			cur = snap.Registration
			arm(cur)
		case <-timer.C:
			if cur == nil {
				continue
			}
			logger.Info("renewing cloud certificate", zap.String("nodeId", cur.NodeID()))
			if err := conn.Renew(ctx); err != nil {
				if time.Now().After(cur.Leaf().NotAfter) {
					logger.Warn("cloud certificate expired before renewal succeeded; re-enrollment required",
						zap.String("nodeId", cur.NodeID()), zap.Error(err))
					continue // do not retry-loop on an expired certificate
				}
				logger.Error("cloud certificate renewal failed; will retry", zap.Error(err))
				// the timer has just fired, so its channel is already drained
				timer.Reset(renewalRetryInterval)
				continue
			}
			// success: conn.Renew broadcasts the new credential; the state-change
			// branch re-arms the timer to the new expiry.
		}
	}
}

// RenewAt returns the time at which the controller will begin renewing leaf —
// roughly two-thirds through its lifetime. It is the point AutoRenew acts on,
// surfaced for display.
func RenewAt(leaf *x509.Certificate) time.Time {
	lifetime := leaf.NotAfter.Sub(leaf.NotBefore)
	return leaf.NotBefore.Add(time.Duration(float64(lifetime) * renewalProgress))
}

// untilRenew returns how long from now until renewal should begin for leaf:
// a negative duration if the certificate has already expired (do not renew),
// zero if renewal is already due, otherwise the time remaining.
func untilRenew(leaf *x509.Certificate, now time.Time) time.Duration {
	if now.After(leaf.NotAfter) {
		return -1
	}
	renewAt := RenewAt(leaf)
	if !now.Before(renewAt) {
		return 0
	}
	return renewAt.Sub(now)
}
