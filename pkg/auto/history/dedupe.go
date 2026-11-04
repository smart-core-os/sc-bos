package history

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
)

type deduper[T proto.Message] struct {
	rec   *recency // also guards last, which the heartbeat and the pull or poll loop both reach
	last  T
	equal cmp.Message
}

func newDeduper[T proto.Message](message cmp.Message, rec *recency) *deduper[T] {
	if message == nil {
		message = cmp.Equal()
	}
	return &deduper[T]{equal: message, rec: rec}
}

// Changed reports whether m should be recorded, and if so remembers it as the last message.
// It is true when m is not equal to the last message, using the provided equal comparator or a default one,
// and also when rec says a reading is due regardless of value:
// maxInterval has passed since the last accepted one, or the last write failed.
// It is false for a polled m whose read began before the last message was accepted, see recency.staleLocked.
func (d *deduper[T]) Changed(ctx context.Context, m T) bool {
	d.rec.mu.Lock()
	defer d.rec.mu.Unlock()
	if d.rec.staleLocked(ctx) {
		return false
	}
	now := time.Now()
	if d.equal(d.last, m) && !d.rec.dueLocked(now) {
		return false
	}

	d.last = m
	d.rec.acceptLocked(now)

	return true
}
