package bms

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/bms/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/modepb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func TestNextRetryDelay(t *testing.T) {
	const base = time.Minute
	const max = 15 * time.Minute

	t.Run("grows exponentially with zero jitter", func(t *testing.T) {
		// jitter 0 => result is exactly half the capped exponential delay.
		cases := []struct {
			attempt int
			wantD   time.Duration // capped exponential delay before jitter
		}{
			{1, 1 * time.Minute},
			{2, 2 * time.Minute},
			{3, 4 * time.Minute},
			{4, 8 * time.Minute},
			{5, 15 * time.Minute}, // 16m capped to 15m
			{6, 15 * time.Minute}, // stays capped
			{20, 15 * time.Minute},
		}
		for _, c := range cases {
			got := nextRetryDelay(base, max, c.attempt, 0)
			if want := c.wantD / 2; got != want {
				t.Errorf("attempt %d: got %v, want %v", c.attempt, got, want)
			}
		}
	})

	t.Run("jitter stays within [d/2, d)", func(t *testing.T) {
		for attempt := 1; attempt <= 6; attempt++ {
			d := nextRetryDelay(base, max, attempt, 0) * 2 // full capped delay
			for _, j := range []float64{0, 0.25, 0.5, 0.75, 0.999999} {
				got := nextRetryDelay(base, max, attempt, j)
				if got < d/2 || got >= d {
					t.Errorf("attempt %d jitter %v: got %v, want in [%v, %v)", attempt, j, got, d/2, d)
				}
			}
		}
	})

	t.Run("attempt below one is treated as one", func(t *testing.T) {
		want := nextRetryDelay(base, max, 1, 0)
		for _, attempt := range []int{0, -1, -100} {
			if got := nextRetryDelay(base, max, attempt, 0); got != want {
				t.Errorf("attempt %d: got %v, want %v", attempt, got, want)
			}
		}
	})

	t.Run("non-positive base falls back to default", func(t *testing.T) {
		got := nextRetryDelay(0, max, 1, 0)
		if want := config.DefaultWriteRetryDelay / 2; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("extreme attempt does not overflow", func(t *testing.T) {
		// a huge attempt count must not panic or wrap to a negative/zero delay.
		got := nextRetryDelay(base, max, 1_000_000, 0)
		if got != max/2 {
			t.Errorf("got %v, want %v", got, max/2)
		}
	})
}

// failToggleActions is an Actions whose writes fail while fail is set, and succeed otherwise.
type failToggleActions struct {
	fail atomic.Bool
}

func (a *failToggleActions) UpdateAirTemperature(context.Context, *airtemperaturepb.UpdateAirTemperatureRequest, *WriteState) error {
	if a.fail.Load() {
		return errors.New("boom")
	}
	return nil
}

func (a *failToggleActions) UpdateModeValues(context.Context, *modepb.UpdateModeValuesRequest, *WriteState) error {
	if a.fail.Load() {
		return errors.New("boom")
	}
	return nil
}

// TestProcessReadStates_Backoff drives the processing loop with a failing write and asserts
// the retry TTL backs off on consecutive failures and resets to the base delay after a success.
func TestProcessReadStates_Backoff(t *testing.T) {
	acts := &failToggleActions{}
	acts.fail.Store(true)

	type doneEvent struct {
		ttl time.Duration
		err error
	}
	done := make(chan doneEvent, 1)

	var mu sync.Mutex
	var curTimer chan time.Time // the most recently created ttl timer

	a := &Auto{
		logger: zap.NewNop(),
		now:    time.Now,
		newTimer: func(time.Duration) (<-chan time.Time, func() bool) {
			ch := make(chan time.Time, 1)
			mu.Lock()
			curTimer = ch
			mu.Unlock()
			return ch, func() bool { return false }
		},
		processDone: func(_ *ReadState, _ *WriteState, ttl time.Duration, err error) {
			done <- doneEvent{ttl: ttl, err: err}
		},
		randFloat: func() float64 { return 0 }, // deterministic: ttl == half the exponential delay
	}

	// a read state that turns occupancy on, forcing mode writes (which fail while acts.fail is set).
	lt := newLogicTester(t)
	lt.controlsOccupancy()
	lt.setOccupied("sensor1")
	// a non-zero WriteEvery guarantees a timer is created on success, so we can drive the next iteration.
	lt.rs.Config.WriteEvery = &jsontypes.Duration{Duration: 30 * time.Minute}
	rs := lt.rs

	readStates := make(chan *ReadState)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := make(chan error, 1)
	go func() { errc <- a.processReadStates(ctx, readStates, acts) }()

	// fireTimer triggers the ttlExpired path, causing the loop to reprocess the last read state.
	fireTimer := func() {
		mu.Lock()
		ch := curTimer
		mu.Unlock()
		ch <- time.Time{}
	}

	// initial process: fails (attempt 1).
	readStates <- rs
	e1 := <-done
	if e1.err == nil {
		t.Fatalf("expected first process to fail")
	}

	// two more failures, TTL should grow each time.
	fireTimer()
	e2 := <-done
	fireTimer()
	e3 := <-done
	if !(e1.ttl < e2.ttl && e2.ttl < e3.ttl) {
		t.Errorf("expected backoff to grow, got %v, %v, %v", e1.ttl, e2.ttl, e3.ttl)
	}

	// a success resets the backoff.
	acts.fail.Store(false)
	fireTimer()
	e4 := <-done
	if e4.err != nil {
		t.Fatalf("expected success, got %v", e4.err)
	}

	// the next failure should be back at the base delay (same as the very first failure).
	acts.fail.Store(true)
	fireTimer()
	e5 := <-done
	if e5.err == nil {
		t.Fatalf("expected failure after reset")
	}
	if e5.ttl != e1.ttl {
		t.Errorf("expected backoff to reset to %v, got %v", e1.ttl, e5.ttl)
	}

	cancel()
	if err := <-errc; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("unexpected loop error: %v", err)
	}
}
