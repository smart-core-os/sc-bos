package dataretention

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// TestPollLoop_FullOnlyWhenWatched verifies the full/cheap update schedule of an
// alwaysPoll loop: full at startup, cheap on ticks while nobody subscribes, full
// immediately when a subscriber connects and on ticks while they stay connected.
func TestPollLoop_FullOnlyWhenWatched(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var fulls []bool
		wantFulls := func(want ...bool) {
			t.Helper()
			mu.Lock()
			defer mu.Unlock()
			if !slices.Equal(fulls, want) {
				t.Errorf("update full args = %v, want %v", fulls, want)
			}
		}

		model := dataretentionpb.NewModel()
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			defer close(done)
			pollLoop(ctx, model, true, func(_ context.Context, full bool) {
				mu.Lock()
				fulls = append(fulls, full)
				mu.Unlock()
			})
		}()

		// Startup update is full; unsubscribed ticks are cheap.
		synctest.Wait()
		wantFulls(true)
		time.Sleep(30 * time.Second)
		synctest.Wait()
		time.Sleep(30 * time.Second)
		synctest.Wait()
		wantFulls(true, false, false)

		// A subscriber connecting triggers an immediate full update, and ticks stay
		// full while they remain connected.
		subCtx, unsubscribe := context.WithCancel(ctx)
		model.PullDataRetention(subCtx)
		synctest.Wait()
		wantFulls(true, false, false, true)
		time.Sleep(30 * time.Second)
		synctest.Wait()
		wantFulls(true, false, false, true, true)

		// After the subscriber disconnects, ticks return to cheap. The first tick after
		// disconnect may still be full (the loop only rechecks subscribers per iteration).
		unsubscribe()
		synctest.Wait()
		time.Sleep(30 * time.Second) // may be full or cheap, ignored
		synctest.Wait()
		mu.Lock()
		fulls = nil
		mu.Unlock()
		time.Sleep(30 * time.Second)
		synctest.Wait()
		wantFulls(false)

		cancel()
		<-done
	})
}

// TestStartPolling_DisposeWaitsForInFlightUpdate verifies that the Undo returned by
// startPolling does not dispose the health check until the poll goroutine has fully
// exited. storageHealth's check is not safe for concurrent use, so an in-flight update
// must never overlap dispose.
func TestStartPolling_DisposeWaitsForInFlightUpdate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		disposed := false
		updateAfterDispose := false

		registry := healthpb.NewRegistry(
			healthpb.WithOnCheckDelete(func(_, _ string) {
				mu.Lock()
				disposed = true
				mu.Unlock()
			}),
		)
		health := &storageHealth{
			checks:  registry.ForOwner("stores"),
			name:    "n/stores/history",
			highPct: 90,
			logger:  zap.NewNop(),
		}

		inUpdate := make(chan struct{}) // closed once the update is in flight
		release := make(chan struct{})  // closes to let the in-flight update return
		var once sync.Once

		model := dataretentionpb.NewModel()
		undo := startPolling(t.Context(), model, true, health, func(ctx context.Context, _ bool) {
			mu.Lock()
			if disposed {
				updateAfterDispose = true
			}
			mu.Unlock()
			health.update(ctx, 50) // creates the check on the first call
			once.Do(func() {
				close(inUpdate)
				<-release
			})
		})

		// The immediate update runs in the poll goroutine and parks on release.
		<-inUpdate
		synctest.Wait()

		// undo cancels the loop, then must block until the in-flight update returns.
		undoDone := make(chan struct{})
		go func() {
			undo()
			close(undoDone)
		}()
		synctest.Wait() // undo has called stop() and is now blocked waiting for the goroutine

		mu.Lock()
		if disposed {
			t.Error("check disposed while an update was still in flight")
		}
		mu.Unlock()
		select {
		case <-undoDone:
			t.Fatal("undo returned before the in-flight update completed")
		default:
		}

		// Let the in-flight update finish; undo should now exit and dispose the check.
		close(release)
		<-undoDone

		mu.Lock()
		defer mu.Unlock()
		if !disposed {
			t.Error("check was not disposed after undo returned")
		}
		if updateAfterDispose {
			t.Error("update ran after the check was disposed")
		}
	})
}
