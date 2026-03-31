package rx

import (
	"context"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

// Val is a reactive value.
type Val[T any] struct {
	m sync.Mutex
	v T
	b *minibus.Bus[T]
}

func NewVal[T any](v T) *Val[T] {
	return &Val[T]{
		v: v,
		b: &minibus.Bus[T]{},
	}
}

func (v *Val[T]) Get() T {
	v.m.Lock()
	defer v.m.Unlock()
	return v.v
}

// Set sets the value of v to val, returning the old value.
func (v *Val[T]) Set(val T) (old T) {
	v.m.Lock()
	defer v.m.Unlock()
	old, v.v = v.v, val
	v.b.Send(context.Background(), v.v)
	return old
}

// Sub returns the current value of v and a channel that will receive updates to the value of v.
func (v *Val[T]) Sub(ctx context.Context) (T, <-chan T) {
	v.m.Lock()
	defer v.m.Unlock()
	val := v.v
	c := v.b.Listen(ctx)
	return val, c
}
