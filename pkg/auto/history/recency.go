package history

import (
	"context"
	"sync"
	"time"
)

// recency tracks when a source last had a reading accepted for storage.
// One is shared by a source's deduper, its heartbeat and storePayloads.
//
// It exists so a gap in history means the source could not be read, rather than that it didn't change:
// a deduper accepts an unchanged reading once maxInterval has passed,
// and the heartbeat asks the source for one when nothing else has arrived in that time.
type recency struct {
	maxInterval time.Duration // <= 0 means no heartbeat: readings are deduped on value alone
	// wake nudges the heartbeat after a failed write, so the reading is retried sooner than maxInterval.
	wake chan struct{}

	mu sync.Mutex
	// lastAt is when a reading was last accepted.
	// It starts at creation, so the first heartbeat comes maxInterval after start rather than at start:
	// a model-backed source answers a Get with an empty reading until it has been set,
	// which would put a bogus zero into cumulative history.
	lastAt time.Time
	// forget is set when a write fails, so the next reading is recorded even if it is equal to the lost one.
	forget bool
}

func newRecency(maxInterval time.Duration) *recency {
	return &recency{
		maxInterval: maxInterval,
		wake:        make(chan struct{}, 1),
		lastAt:      time.Now(),
	}
}

// accept notes that a reading has been accepted for storage, unless it is stale.
// It reports whether the reading was accepted.
func (r *recency) accept(ctx context.Context) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.staleLocked(ctx) {
		return false
	}
	r.acceptLocked(time.Now())
	return true
}

func (r *recency) acceptLocked(now time.Time) {
	r.lastAt = now
	r.forget = false
}

// dueLocked reports whether a reading should be accepted regardless of its value.
func (r *recency) dueLocked(now time.Time) bool {
	return r.forget || (r.maxInterval > 0 && now.Sub(r.lastAt) >= r.maxInterval)
}

// staleLocked reports whether ctx is for a read that began before the last reading was accepted.
// That reading came from elsewhere, a pull or another poll, and is at least as new,
// so recording this one after it could put an older value after a newer one: a backwards step for a meter.
func (r *recency) staleLocked(ctx context.Context) bool {
	start, ok := ctx.Value(readStartKey{}).(time.Time)
	return ok && r.lastAt.After(start)
}

type readStartKey struct{}

// withReadStart wraps poll so readings it returns can be recognised as stale by recency.
func withReadStart(poll getter) getter {
	return func(ctx context.Context, changes chan<- []byte) error {
		return poll(context.WithValue(ctx, readStartKey{}, time.Now()), changes)
	}
}

// failed notes that an accepted reading could not be stored.
func (r *recency) failed() {
	r.mu.Lock()
	r.forget = true
	r.mu.Unlock()
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// next returns when a heartbeat is next due, if no reading is accepted before then.
func (r *recency) next() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastAt.Add(r.maxInterval)
}
