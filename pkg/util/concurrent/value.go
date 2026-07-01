package concurrent

import (
	"context"
	"sync"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

type Value[T any] struct {
	m        sync.RWMutex
	value    T
	modified time.Time
	bus      minibus.Bus[ValueEvent[T]]
}

func NewValue[T any](initial T) *Value[T] {
	return &Value[T]{
		value:    initial,
		modified: time.Now(),
	}
}

func (v *Value[T]) Get() (value T, modified time.Time) {
	v.m.RLock()
	defer v.m.RUnlock()

	return v.value, v.modified
}

func (v *Value[T]) Set(ctx context.Context, value T) (old T, ok bool) {
	v.m.Lock()
	old = v.value
	v.value = value
	timestamp := time.Now()
	v.modified = timestamp
	v.m.Unlock()

	v.bus.Send(ctx, ValueEvent[T]{
		Timestamp: timestamp,
		Old:       old,
		New:       value,
	})
	// ok reports whether the change was distributed without ctx being cancelled.
	ok = ctx.Err() == nil
	return
}

// Changes returns the current value and a channel of future changes.
// With backpressure, changes are sent directly over the returned channel and a slow receiver will block senders.
// Without backpressure, the bufferSize most recent changes are buffered and older ones are silently discarded if
// the receiver does not keep up; bufferSize must be 1 or more.
func (v *Value[T]) Changes(ctx context.Context, backpressure bool, bufferSize int) (value T, changes <-chan ValueEvent[T]) {
	v.m.RLock()
	defer v.m.RUnlock()

	ch := v.bus.Listen(ctx)
	if !backpressure {
		ch = BreakBackpressureBuffered(ch, bufferSize)
	}
	return v.value, ch
}

type ValueEvent[T any] struct {
	Timestamp time.Time
	Old       T
	New       T
}
