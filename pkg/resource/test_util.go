package resource

import (
	"testing"
	"time"
)

func waitForChan[T any](t *testing.T, c <-chan T, wait time.Duration) T {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case v := <-c:
		return v
	case <-timer.C:
		t.Fatalf("timeout waiting for chan")
		var zero T
		return zero
	}
}

func noEmitWithin[T any](t *testing.T, c <-chan T, wait time.Duration) {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case v := <-c:
		t.Fatalf("Not expecting a value on chan, got %v", v)
	case <-timer.C:
	}
}
