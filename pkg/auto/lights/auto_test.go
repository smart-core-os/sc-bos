package lights

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

var errFailedBrightnessUpdate = errors.New("failed to update brightness this time")

// TestPirsTurnLightsOn drives the automation through a sequence of occupancy
// changes, time-based refreshes, and retry/back-off scenarios.
//
// It runs inside a synctest bubble so the automation's timers and goroutines
// are scheduled against a deterministic fake clock. This removes the races that
// previously made the test flaky: rather than hand-rolling a clock and relying
// on the automation and test goroutines alternating in lock-step over an
// unbuffered channel, we let the automation settle with synctest.Wait and then
// inspect the last completed process, and advance timers explicitly with
// time.Sleep.
func TestPirsTurnLightsOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// epoch is the bubble's start time. Occupancy StateChangeTimes are
		// expressed relative to it (as the old hand-rolled clock did relative to
		// time.Unix(0,0)), so "N minutes ago" keeps its meaning as the fake clock
		// advances through the test.
		epoch := time.Now()

		// we update this to send messages to the automation
		pir01 := occupancysensorpb.NewModel()
		pir02 := occupancysensorpb.NewModel()
		rootNode := node.New("test")
		rootNode.Announce("pir01",
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(pir01))),
			node.HasTrait(trait.OccupancySensor),
		)
		rootNode.Announce("pir02",
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(pir02))),
			node.HasTrait(trait.OccupancySensor),
		)

		testActions := newTestActions(t)

		loggerConfig := zap.NewDevelopmentConfig()
		loggerConfig.DisableStacktrace = true
		logger, _ := loggerConfig.Build()
		logger = logger.Named(fmt.Sprintf("%x", rand.Uint64())) // helps with --count>1 tests
		automation := PirsTurnLightsOn(rootNode, logger)
		automation.makeActions = func(_ node.ClientConner) actions { return testActions }
		automation.autoStartTime = epoch
		// newTimer is left as the default time.NewTimer; synctest fakes it.

		cfg := config.Default()
		cfg.Now = time.Now
		cfg.OccupancySensors = []deviceName{"pir01", "pir02"}
		cfg.Lights = []deviceName{"light01", "light02"}
		cfg.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
		cfg.RefreshEvery = &jsontypes.Duration{Duration: 8 * time.Minute}
		cfg.LogTriggers = true
		cfg.LogEmptyChanges = true

		rec := newProcessRecorder()
		automation.processComplete = rec.record

		if err := automation.Start(context.Background()); err != nil {
			t.Fatalf("Start: %v", err)
		}
		// Stop the automation so its goroutines (subscriptions, processing loop)
		// exit before the synctest bubble ends; otherwise synctest panics about
		// leftover blocked goroutines.
		t.Cleanup(func() {
			_ = automation.Stop()
			synctest.Wait()
		})

		if err := automation.configure(cfg); err != nil {
			t.Fatalf("Configure: %v", err)
		}

		// check setting occupied on one PIR causes the lights to come on
		_, _ = pir01.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED})
		e := rec.take(t)
		assertOccupancy(t, e.readState, "pir01", occupancysensorpb.Occupancy_OCCUPIED)
		assertNoErrAndTtl(t, e.ttl, e.err, cfg.RefreshEvery.Duration)
		testActions.assertNextBrightnessUpdates(100, "light01", "light02")

		// check that setting occupied on the other PIR does nothing
		_, _ = pir02.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED})
		e = rec.take(t)
		assertOccupancy(t, e.readState, "pir02", occupancysensorpb.Occupancy_OCCUPIED)
		assertNoErrAndTtl(t, e.ttl, e.err, cfg.RefreshEvery.Duration)
		testActions.assertNoMoreCalls()

		// check that making both PIRs unoccupied doesn't do anything, but then does
		_, _ = pir01.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(epoch.Add(-3 * time.Minute))})
		_, _ = pir02.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(epoch.Add(-8 * time.Minute))})
		e = rec.take(t)
		assertOccupancy(t, e.readState, "pir01", occupancysensorpb.Occupancy_UNOCCUPIED)
		assertOccupancy(t, e.readState, "pir02", occupancysensorpb.Occupancy_UNOCCUPIED)
		if want := 7 * time.Minute; e.ttl != want {
			t.Fatalf("TTL want %v, got %v", want, e.ttl)
		}
		if e.err != nil {
			t.Fatalf("Got error %v", e.err)
		}
		// trigger the timer
		time.Sleep(e.ttl)
		e = rec.take(t) // no state change, only time change
		assertNoErrAndTtl(t, e.ttl, e.err, cfg.RefreshEvery.Duration)
		testActions.assertNextBrightnessUpdates(0, "light01", "light02")

		// test 1 retry
		testActions.nextCallReturnsError(errFailedBrightnessUpdate, "light01")
		_, _ = pir01.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED})
		e = rec.take(t)
		assertOccupancy(t, e.readState, "pir01", occupancysensorpb.Occupancy_OCCUPIED)
		// jitter is set to ±0.2
		assertErrorAndTtl(t, e.ttl, e.err, cfg.OnProcessError.BackOffMultiplier.Duration*8/10, errFailedBrightnessUpdate)
		testActions.assertNextBrightnessUpdates(100, "light01", "light02")

		// advance to the retry deadline to force a replay
		time.Sleep(e.ttl)
		e = rec.take(t)
		assertOccupancy(t, e.readState, "pir01", occupancysensorpb.Occupancy_OCCUPIED)
		assertNoErrAndTtl(t, e.ttl, e.err, cfg.RefreshEvery.Duration)
		// it works after the retry
		testActions.assertNextBrightnessUpdates(100, "light01") // light02 is cached, so no update

		// testing retries getting cancelled after max attempts
		_, _ = pir01.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(epoch.Add(-3 * time.Minute))})
		_, _ = pir02.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(epoch.Add(-3 * time.Minute))})
		e = rec.take(t)
		assertOccupancy(t, e.readState, "pir01", occupancysensorpb.Occupancy_UNOCCUPIED)
		assertOccupancy(t, e.readState, "pir02", occupancysensorpb.Occupancy_UNOCCUPIED)
		assertNoErrAndTtl(t, e.ttl, e.err, cfg.RefreshEvery.Duration)
		testActions.assertNextBrightnessUpdates(0, "light01", "light02")

		testActions.nextCallReturnsError(fmt.Errorf("attempt 1: %w", errFailedBrightnessUpdate), "light01", "light02")
		_, _ = pir01.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED})
		e = rec.take(t)
		assertOccupancy(t, e.readState, "pir01", occupancysensorpb.Occupancy_OCCUPIED)
		assertErrorAndTtl(t, e.ttl, e.err, (cfg.OnProcessError.BackOffMultiplier.Duration*8)/10, errFailedBrightnessUpdate)
		testActions.assertNextBrightnessUpdates(100, "light01", "light02")

		// second try
		testActions.nextCallReturnsError(fmt.Errorf("attempt 2: %w", errFailedBrightnessUpdate), "light01", "light02")
		time.Sleep(e.ttl) // advance to the retry deadline to force a replay
		e = rec.take(t)   // no state change, only time change
		assertErrorAndTtl(t, e.ttl, e.err, 2*(cfg.OnProcessError.BackOffMultiplier.Duration*8)/10, errFailedBrightnessUpdate)
		testActions.assertNextBrightnessUpdates(100, "light01", "light02")

		// third try and is cancelled
		testActions.nextCallReturnsError(fmt.Errorf("attempt 3: %w", errFailedBrightnessUpdate), "light01", "light02")
		time.Sleep(e.ttl) // advance to the retry deadline to force a replay
		e = rec.take(t)   // no state change, only time change
		assertErrorAndTtl(t, e.ttl, e.err, -time.Nanosecond, errFailedBrightnessUpdate)
		testActions.assertNextBrightnessUpdates(100, "light01", "light02")
		// ensure we have effectively cancelled reprocessing
		synctest.Wait()
		testActions.assertNoMoreCalls()
	})
}

// processCompleteEvent captures the arguments of a single processComplete callback.
type processCompleteEvent struct {
	ttl        time.Duration
	err        error
	readState  *ReadState
	writeState *WriteState
}

// processRecorder records processComplete callbacks from the automation so the
// test can inspect the most recent completed process after letting the
// automation settle with synctest.Wait.
type processRecorder struct {
	mu    sync.Mutex
	last  processCompleteEvent
	count int // number of processComplete callbacks received
	taken int // count value at the most recent take
}

func newProcessRecorder() *processRecorder {
	return &processRecorder{}
}

func (r *processRecorder) record(ttl time.Duration, err error, readState *ReadState, writeState *WriteState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.last = processCompleteEvent{ttl: ttl, err: err, readState: readState, writeState: writeState}
	r.count++
}

// take waits for the automation to settle, then returns the most recent
// completed process. It fails the test if no new process has completed since
// the previous take.
func (r *processRecorder) take(t *testing.T) processCompleteEvent {
	t.Helper()
	synctest.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count == r.taken {
		t.Fatalf("expected a process to complete, but none did")
	}
	r.taken = r.count
	return r.last
}

func assertOccupancy(t *testing.T, state *ReadState, name string, want occupancysensorpb.Occupancy_State) {
	t.Helper()
	o, ok := state.Occupancy[name]
	if !ok {
		t.Fatalf("occupancy for %q not present in read state", name)
	}
	if o.State != want {
		t.Fatalf("occupancy for %q want %v, got %v", name, want, o.State)
	}
}
