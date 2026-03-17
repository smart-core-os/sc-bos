package resources

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// ValueChange is like resource.ValueChange, but with a generic type T.
type ValueChange[T proto.Message] struct {
	Value      T
	ChangeTime time.Time
	// LastSeedValue will be true if this change is the last change as part of the seed values.
	LastSeedValue bool
}

// PullValue converts a stream of resource.ValueChange into a stream of ValueChange[T].
//
// Typical use looks like this:
//
//	func (m *Model) PullMyValue(ctx context.Context, opts ...resource.ReadOption) <-chan ValueChange[MyProtoMessage] {
//	    // m.myValue is a *resource.Value of type MyProtoMessage
//	    return resources.PullValue[MyProtoMessage](ctx, m.myValue.Pull(ctx, opts...))
//	}
func PullValue[T proto.Message](ctx context.Context, stream <-chan *resource.ValueChange) <-chan ValueChange[T] {
	out := make(chan ValueChange[T])
	go func() {
		defer close(out)
		for v := range stream {
			select {
			case <-ctx.Done():
				return
			case out <- ValueChange[T]{
				Value:         v.Value.(T),
				ChangeTime:    v.ChangeTime,
				LastSeedValue: v.LastSeedValue,
			}:
			}
		}
	}()
	return out
}

// CollectionChange is like resource.CollectionChange, but with a generic type T.
type CollectionChange[T proto.Message] struct {
	Id         string
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   T
	NewValue   T
	// LastSeedValue will be true if this change is the last change as part of the seed values.
	LastSeedValue bool
}

// PullCollection converts a stream of resource.CollectionChange into a stream of CollectionChange[T].
//
// Typical use looks like this:
//
//	func (m *Model) PullMyCollection(ctx context.Context, opts ...resource.ReadOption) <-chan CollectionChange[MyProtoMessage] {
//	    // m.myCollection is a *resource.Collection of type MyProtoMessage
//	    return resources.PullCollection[MyProtoMessage](ctx, m.myCollection.Pull(ctx, opts...))
//	}
func PullCollection[T proto.Message](ctx context.Context, stream <-chan *resource.CollectionChange) <-chan CollectionChange[T] {
	out := make(chan CollectionChange[T])
	go func() {
		defer close(out)
		for v := range stream {
			c := CollectionChange[T]{
				Id:            v.Id,
				ChangeTime:    v.ChangeTime,
				ChangeType:    v.ChangeType,
				LastSeedValue: v.LastSeedValue,
			}
			if v.OldValue != nil {
				c.OldValue = v.OldValue.(T)
			}
			if v.NewValue != nil {
				c.NewValue = v.NewValue.(T)
			}
			select {
			case <-ctx.Done():
				return
			case out <- c:
			}
		}
	}()
	return out
}
