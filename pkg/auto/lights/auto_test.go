package lights

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

var errFailedBrightnessUpdate = errors.New("failed to update brightness this time")

func TestPirsTurnLightsOn(t *testing.T) {
	// we update this to send messages to the automation
	pir01 := occupancysensorpb.NewModel()
	pir02 := occupancysensorpb.NewModel()
	rootNode := node.New("test")
	rootNode.Announce("pir01", node.HasTrait(trait.OccupancySensor, node.WithClients(occupancysensorpb.WrapApi(occupancysensorpb.NewModelServer(pir01)))))
	rootNode.Announce("pir02", node.HasTrait(trait.OccupancySensor, node.WithClients(occupancysensorpb.WrapApi(occupancysensorpb.NewModelServer(pir02)))))

	testActions := newTestActions(t)

	clock := newTestClock(t)

	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.DisableStacktrace = true
	logger, _ := loggerConfig.Build()                       // the test is sometimes failing, this might help with debugging
	logger = logger.Named(fmt.Sprintf("%x", rand.Uint64())) // helps with --count>1 tests
	automation := PirsTurnLightsOn(rootNode, logger)
	automation.makeActions = func(_ node.ClientConner) actions { return testActions }
	automation.autoStartTime = clock.now
	automation.newTimer = clock.newTimer

	cfg := config.Default()
	cfg.Now = clock.nowFunc
	cfg.OccupancySensors = []deviceName{"pir01", "pir02"}
	cfg.Lights = []deviceName{"light01", "light02"}
	cfg.UnoccupiedOffDelay = jsontypes.Duration{Duration: 10 * time.Minute}
	cfg.RefreshEvery = &jsontypes.Duration{Duration: 8 * time.Minute}
	cfg.LogTriggers = true
	cfg.LogEmptyChanges = true

	type processCompleteEvent struct {
		ttl        time.Duration
		err        error
		readState  *ReadState
		writeState *WriteState
	}
	processCompleteC := make(chan processCompleteEvent)
	automation.processComplete = func(ttl time.Duration, err error, readState *ReadState, writeState *WriteState) {
		processCompleteC <- processCompleteEvent{
			ttl:        ttl,
			err:        err,
			readState:  readState,
			writeState: writeState,
		}
	}
	const stateWaitTime = 10 * time.Second
	waitForState := func(test func(state *ReadState) bool) (time.Duration, error) {
		t.Helper()
		timeout := time.NewTimer(stateWaitTime)
		for {
			select {
			case <-timeout.C:
				t.Fatalf("timeout waiting for state")
				return 0, nil
			case e := <-processCompleteC:
				if test(e.readState) {
					return e.ttl, e.err
				}
			}
		}
	}

	if err := automation.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := automation.configure(cfg); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	// check setting occupied on one PIR causes the lights to come on
	_, _ = pir01.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_OCCUPIED})
	ttl, err := waitForState(func(state *ReadState) bool {
		o, ok := state.Occupancy["pir01"]
		if !ok {
			return false
		}
		return o.State == traits.Occupancy_OCCUPIED
	})
	assertNoErrAndTtl(t, ttl, err, cfg.RefreshEvery.Duration)

	testActions.assertNextBrightnessUpdates(100, "light01", "light02")

	// check that setting occupied on the other PIR does nothing
	_, _ = pir02.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_OCCUPIED})
	ttl, err = waitForState(func(state *ReadState) bool {
		o, ok := state.Occupancy["pir02"]
		if !ok {
			return false
		}
		return o.State == traits.Occupancy_OCCUPIED
	})
	assertNoErrAndTtl(t, ttl, err, cfg.RefreshEvery.Duration)
	testActions.assertNoMoreCalls()

	// check that making both PIRs unoccupied doesn't do anything, but then does
	_, _ = pir01.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(0, 0).Add(-3 * time.Minute))})
	_, _ = pir02.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(0, 0).Add(-8 * time.Minute))})
	ttl, err = waitForState(func(state *ReadState) bool {
		o01, ok01 := state.Occupancy["pir01"]
		o02, ok02 := state.Occupancy["pir02"]
		if !ok01 || !ok02 {
			return false
		}
		return o01.State == traits.Occupancy_UNOCCUPIED && o02.State == traits.Occupancy_UNOCCUPIED
	})
	if want := 7 * time.Minute; ttl != want {
		t.Fatalf("TTL want %v, got %v", want, ttl)
	}
	if err != nil {
		t.Fatalf("Got error %v", err)
	}
	// trigger the timer
	clock.advance(7 * time.Minute)
	ttl, err = waitForState(func(state *ReadState) bool {
		return true // no state change, only time change
	})
	assertNoErrAndTtl(t, ttl, err, cfg.RefreshEvery.Duration)
	testActions.assertNextBrightnessUpdates(0, "light01", "light02")

	// test 1 retry
	testActions.nextCallReturnsError(errFailedBrightnessUpdate, "light01")
	_, _ = pir01.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_OCCUPIED})
	ttl, err = waitForState(func(state *ReadState) bool {
		o01, ok01 := state.Occupancy["pir01"]
		if !ok01 {
			return false
		}
		return o01.State == traits.Occupancy_OCCUPIED
	})
	// jitter is set to ±0.2
	assertErrorAndTtl(t, ttl, err, cfg.OnProcessError.BackOffMultiplier.Duration*8/10, errFailedBrightnessUpdate)
	testActions.assertNextBrightnessUpdates(100, "light01", "light02")

	// since newTimer is intercepted by this test, we force a replay here
	clock.advance(500 * time.Millisecond)
	ttl, err = waitForState(func(state *ReadState) bool {
		o01, ok01 := state.Occupancy["pir01"]
		if !ok01 {
			return false
		}
		return o01.State == traits.Occupancy_OCCUPIED
	})
	assertNoErrAndTtl(t, ttl, err, cfg.RefreshEvery.Duration)
	// it works after the retry
	testActions.assertNextBrightnessUpdates(100, "light01") // light02 is cached, so no update

	// testing retries getting cancelled after max attempts
	_, _ = pir01.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(0, 0).Add(-3 * time.Minute))})
	_, _ = pir02.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(0, 0).Add(-3 * time.Minute))})
	ttl, err = waitForState(func(state *ReadState) bool {
		o01, ok01 := state.Occupancy["pir01"]
		o02, ok02 := state.Occupancy["pir02"]
		if !ok01 || !ok02 {
			return false
		}
		return o01.State == traits.Occupancy_UNOCCUPIED && o02.State == traits.Occupancy_UNOCCUPIED
	})
	assertNoErrAndTtl(t, ttl, err, cfg.RefreshEvery.Duration)
	testActions.assertNextBrightnessUpdates(0, "light01", "light02")

	testActions.nextCallReturnsError(fmt.Errorf("attempt 1: %w", errFailedBrightnessUpdate), "light01", "light02")
	_, _ = pir01.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_OCCUPIED})
	ttl, err = waitForState(func(state *ReadState) bool {
		o01, ok01 := state.Occupancy["pir01"]
		if !ok01 {
			return false
		}
		return o01.State == traits.Occupancy_OCCUPIED
	})
	assertErrorAndTtl(t, ttl, err, cfg.OnProcessError.BackOffMultiplier.Duration*8/10, errFailedBrightnessUpdate)
	testActions.assertNextBrightnessUpdates(100, "light01", "light02")

	// second try
	testActions.nextCallReturnsError(fmt.Errorf("attempt 2: %w", errFailedBrightnessUpdate), "light01", "light02")
	// since newTimer is intercepted by this test, we force a replay here
	clock.advance(500 * time.Millisecond)
	ttl, err = waitForState(func(state *ReadState) bool {
		return true // no state change, only time change
	})
	assertErrorAndTtl(t, ttl, err, 2*cfg.OnProcessError.BackOffMultiplier.Duration*(8/10), errFailedBrightnessUpdate)
	testActions.assertNextBrightnessUpdates(100, "light01", "light02")

	// third try and is cancelled
	testActions.nextCallReturnsError(fmt.Errorf("attempt 3: %w", errFailedBrightnessUpdate), "light01", "light02")
	// since newTimer is intercepted by this test, we force a replay here
	clock.advance(500 * time.Millisecond)
	ttl, err = waitForState(func(state *ReadState) bool {
		return true // no state change, only time change
	})
	assertErrorAndTtl(t, ttl, err, -time.Nanosecond, errFailedBrightnessUpdate)
	testActions.assertNextBrightnessUpdates(100, "light01", "light02")
	// ensure we have effectively cancelled reprocessing
	testActions.assertNoMoreCalls()
}

type testClock struct {
	t   *testing.T
	mu  sync.Mutex
	now time.Time
	c   chan time.Time
}

func newTestClock(t *testing.T) *testClock {
	t.Helper()
	return &testClock{
		t:   t,
		now: time.Unix(0, 0),
	}
}

func (tc *testClock) nowFunc() time.Time {
	tc.t.Helper()
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.now
}

func (tc *testClock) newTimer(_ time.Duration) (<-chan time.Time, func() bool) {
	tc.t.Helper()
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.c != nil {
		tc.t.Fatalf("newTimer called multiple times without Stop being called")
	}
	c := make(chan time.Time, 1)
	tc.c = c
	return c, func() bool {
		tc.t.Helper()
		tc.mu.Lock()
		defer tc.mu.Unlock()
		if tc.c != c {
			return true
		}
		tc.c = nil
		return true
	}
}

func (tc *testClock) advance(d time.Duration) {
	tc.t.Helper()
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.c == nil {
		tc.t.Fatalf("advance called without newTimer being called")
	}
	tc.now = tc.now.Add(d)
	select {
	case tc.c <- tc.now:
	default:
		tc.t.Fatalf("advance called but no goroutine is waiting for the timer")
	}
}
