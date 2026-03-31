package pull

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/util/chans"
)

func TestOrPoll(t *testing.T) {
	const delay = 100 * time.Millisecond // used to wait for the blocking go routine

	t.Run("pull", func(t *testing.T) {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(stop)

		clock := clockwork.NewFakeClock()
		pull, get := testGetters(t)

		res := make(chan error, 1)
		go func() {
			res <- OrPoll(ctx, pull.Getter, get.Getter, withClock(clock))
		}()

		pull.AssertCalled()
		get.AssertNotCalled()
		if err := chans.IsEmptyWithin(res, delay); err != nil {
			t.Errorf("expected no result, got %v", err)
		}
	})

	t.Run("get", func(t *testing.T) {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(stop)

		clock := clockwork.NewFakeClock()
		pull, get := testGetters(t)

		res := make(chan error, 1)
		go func() {
			res <- OrPoll(ctx, pull.Getter, get.Getter, withClock(clock), WithPullFallbackJitter(0))
		}()

		pull.Return(status.Error(codes.Unimplemented, "not implemented"))
		returns := get.AssertCalled()
		if err := chans.IsEmptyWithin(res, delay); err != nil {
			t.Errorf("expected no result, got %v", err)
		}

		// check the next get is called after poll delay (and not before)
		returns <- nil
		pull.Return(status.Error(codes.Unimplemented, "not implemented")) // double check
		get.AssertNotCalled()
		if err := clock.BlockUntilContext(ctx, 1); err != nil {
			t.Errorf("expected wait on clock, but got %v", err)
		}
		clock.Advance(DefaultPollDelay)
		get.AssertCalled()
	})

	t.Run("pull retry", func(t *testing.T) {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(stop)

		clock := clockwork.NewFakeClock()
		pull, get := testGetters(t)

		res := make(chan error, 1)
		go func() {
			res <- OrPoll(ctx, pull.Getter, get.Getter, withClock(clock), WithPullFallbackJitter(0))
		}()

		pull.Return(status.Error(codes.NotFound, "not found"))
		if err := clock.BlockUntilContext(ctx, 1); err != nil {
			t.Errorf("expected wait on clock, but got %v", err)
		}
		pull.AssertNotCalled() // should wait for retry delay
		clock.Advance(DefaultRetryInit)
		pull.AssertCalled()
	})

	t.Run("get retry", func(t *testing.T) {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(stop)

		clock := clockwork.NewFakeClock()
		pull, get := testGetters(t)

		res := make(chan error, 1)
		go func() {
			res <- OrPoll(ctx, pull.Getter, get.Getter, withClock(clock), WithPullFallbackJitter(0))
		}()

		pull.Return(status.Error(codes.Unimplemented, "not implemented"))
		get.Return(status.Error(codes.NotFound, "not found"))
		pull.Return(status.Error(codes.Unimplemented, "not implemented: retry"))
		if err := clock.BlockUntilContext(ctx, 1); err != nil {
			t.Errorf("expected wait on clock, but got %v", err)
		}
		get.AssertNotCalled() // should wait for retry delay
		clock.Advance(DefaultPollDelay)
		get.AssertNotCalled()               // on error, we increase the poll delay
		clock.Advance(DefaultPollDelay / 2) // add on the remaining extra time
		get.AssertCalled()
	})

	// pull and get report unimplemented until a delay, then pull reports ok
	t.Run("delayed support", func(t *testing.T) {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(stop)

		clock := clockwork.NewFakeClock()
		pull, get := testGetters(t)

		res := make(chan error, 1)
		go func() {
			res <- OrPoll(ctx, pull.Getter, get.Getter, withClock(clock), WithPullFallbackJitter(0))
		}()

		pull.Return(status.Error(codes.Unimplemented, "not implemented"))
		get.Return(status.Error(codes.Unimplemented, "not implemented"))
		if err := clock.BlockUntilContext(ctx, 1); err != nil {
			t.Errorf("expected wait on clock, but got %v", err)
		}
		pull.AssertNotCalled()
		get.AssertNotCalled()
		clock.Advance(DefaultRetryInit)
		pull.AssertCalled()
	})

	// pull reports unimplemented, get reports ok, pull then reports ok
	t.Run("racy support", func(t *testing.T) {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(stop)

		clock := clockwork.NewFakeClock()
		pull, get := testGetters(t)

		res := make(chan error, 1)
		go func() {
			res <- OrPoll(ctx, pull.Getter, get.Getter, withClock(clock), WithPullFallbackJitter(0))
		}()

		pull.Return(status.Error(codes.Unimplemented, "not implemented"))
		get.Return(nil)
		pull.AssertCalled()
		get.AssertNotCalled()
	})
}

func testGetters(t *testing.T) (pull, get *testGetter) {
	return newTestGetter(t, "pull"), newTestGetter(t, "get")
}

func newTestGetter(t *testing.T, name string) *testGetter {
	return &testGetter{t: t, name: name, calls: make(chan chan error)}
}

// testGetter is a Getter function with utilities for testing.
type testGetter struct {
	t         *testing.T
	name      string
	calls     chan chan error
	closeOnce sync.Once
}

func (g *testGetter) Getter(ctx context.Context) error {
	res := make(chan error, 1)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case g.calls <- res:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-res:
			return err
		}
	}
}

func (g *testGetter) Return(err error) {
	g.AssertCalled() <- err
}

func (g *testGetter) AssertCalled() chan<- error {
	g.t.Helper()
	if res, err := chans.RecvWithin(g.calls, time.Second); err != nil {
		g.t.Errorf("%s expected called, but was not", g.name)
		return nil
	} else {
		return res
	}
}

func (g *testGetter) AssertNotCalled() {
	g.t.Helper()
	if err := chans.IsEmptyWithin(g.calls, 10*time.Millisecond); err != nil {
		g.t.Errorf("%s expected not called, but was", g.name)
	}
}
