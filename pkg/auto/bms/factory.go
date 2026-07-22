package bms

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/bms/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
)

var Factory auto.Factory = factory{}

const AutoType = "bms"

type factory struct{}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &Auto{
		logger:  services.Logger.Named(AutoType),
		clients: services.Node,
	}
	a.Service = service.New(a.applyConfig,
		service.WithParser(config.ReadBytes),
		service.WithOnStop[config.Root](func() {
			if a.stop != nil {
				a.stop()
			}
		}))

	return a
}

type Auto struct {
	*service.Service[config.Root]
	logger  *zap.Logger
	clients node.ClientConner

	setupOnce     sync.Once // reset on stop
	setupErr      error
	configChanges chan<- config.Root

	stop context.CancelFunc

	// test helpers
	now           func() time.Time
	clientActions func(clientConner node.ClientConner) Actions
	newTimer      func(duration time.Duration) (<-chan time.Time, func() bool)
	processDone   func(readState *ReadState, writeState *WriteState, ttl time.Duration, err error)
	randFloat     func() float64 // returns a value in [0.0,1.0), used to jitter retry backoff
}

func (a *Auto) applyConfig(ctx context.Context, cfg config.Root) error {
	a.setTestHelperFuncs()

	actions := a.clientActions(a.clients)
	cfgChanges, err := a.setup(actions)
	if err != nil {
		return err
	}
	return chans.SendContext(ctx, cfgChanges, cfg)
}

func (a *Auto) setup(actions Actions) (chan<- config.Root, error) {
	a.setupOnce.Do(func() {
		patches := make(chan Patcher, 10)
		cfgChanges := make(chan config.Root, 1)

		a.configChanges = cfgChanges
		ctx, stop := context.WithCancel(context.Background())
		a.stop = func() {
			stop()
			a.configChanges = nil
			a.setupErr = nil
			a.setupOnce = sync.Once{} // allow setup to run again
		}

		readStates := make(chan *ReadState)
		initialState := NewReadState()
		initialState.Now = a.now
		initialState.StartTime = a.now()

		group, ctx := errgroup.WithContext(ctx)
		// watch for config changes and setup the relevant patch sources to read from configured devices
		group.Go(func() error {
			defer close(cfgChanges)
			err := a.setupPatchers(ctx, cfgChanges, patches)
			if err != nil {
				return fmt.Errorf("setupPatchers: %w", err)
			}
			return err
		})

		// process patches into a read state
		group.Go(func() error {
			defer close(readStates)
			err := processPatches(ctx, initialState, patches, readStates)
			if err != nil {
				return fmt.Errorf("processPatches: %w", err)
			}
			return err
		})

		// process read state into side effects
		group.Go(func() error {
			err := a.processReadStates(ctx, readStates, actions)
			if err != nil {
				return fmt.Errorf("processReadStates: %w", err)
			}
			return err
		})

		go func() {
			err := group.Wait()
			if errors.Is(err, context.Canceled) {
				return
			}
			a.logger.Error("background task(s) failed", zap.Error(err))
		}()
	})
	return a.configChanges, a.setupErr
}

// processReadStates reads ReadState from a channel and analyses each entry deciding what should be changed.
// This function backs off to processReadState which has the actual logic for what to do given a certain state,
// this function handles the channel management, retry logic, TTL on decisions, and all that type of thing.
func (a *Auto) processReadStates(ctx context.Context, readStates <-chan *ReadState, actions Actions) error {
	// the below are the innards of time.Timer, but expanded so we can stop/select on them even if
	// we don't have a timer active right now

	var ttlExpired <-chan time.Time
	cancelTtlTimer := func() bool { return false }

	// writeState is only accessed from this go routine.
	writeState := NewWriteState()
	writeState.Now = a.now

	// consecutiveFailures counts how many times processing has failed in a row.
	// It grows the retry backoff and resets to 0 on the next successful process.
	var consecutiveFailures int

	var lastProcessedState *ReadState
	processStateFn := func(readState *ReadState) error {
		cancelTtlTimer()
		logger := a.logger.With(zap.String("auto", readState.Config.Name))

		if readState.Config.LogReads {
			logReads(logger, lastProcessedState, readState)
			lastProcessedState = readState
		}

		actions := actions
		if readState.Config.DryRun {
			actions = NilActions{}
		}
		if readState.Config.LogDeviceWrites {
			actions = LogActions(actions, logger)
		}
		actions, actionCounts := CountActions(actions)
		actions = CacheWriteAction(actions, readState.Config.WriteCacheExpiry.Or(config.DefaultWriteCacheExpiry))

		writeState.Before()
		writeState.CopyFromReadState(readState)
		ttl, err := processReadState(ctx, readState, writeState, actions)
		writeState.After()

		refreshEvery := readState.Config.WriteEvery.Or(config.DefaultWriteEvery)
		switch {
		case ctx.Err() != nil:
			return ctx.Err()
		case err != nil:
			// Back off exponentially (with jitter) while processing keeps failing,
			// so an unreachable device isn't hammered at a constant rate.
			consecutiveFailures++
			base := readState.Config.WriteRetryDelay.Or(config.DefaultWriteRetryDelay)
			maxDelay := readState.Config.WriteRetryMaxDelay.Or(config.DefaultWriteRetryMaxDelay)
			ttl = nextRetryDelay(base, maxDelay, consecutiveFailures, a.randFloat())
		default:
			// processing succeeded, reset the backoff
			consecutiveFailures = 0
			if refreshEvery > 0 && (ttl <= 0 || ttl > refreshEvery) {
				// ensure it's not too long before we wake up,
				// so external changes don't stick around forever
				ttl = refreshEvery
			}
		}

		// Setup ttl for the transformed model.
		// After this time it should be recalculated.
		if ttl > 0 {
			ttlExpired, cancelTtlTimer = a.newTimer(ttl)
		}

		// log side effects
		logWrites(logger, readState, writeState, actionCounts, ttl, err)

		// notify for testing
		a.processDone(readState, writeState, ttl, err)

		return nil
	}

	var lastReadState *ReadState

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case readState := <-readStates:
			lastReadState = readState
			err := processStateFn(readState)
			if err != nil {
				return err
			}
		case <-ttlExpired:
			err := processStateFn(lastReadState)
			if err != nil {
				return err
			}
		}
	}
}

// nextRetryDelay computes the delay before the next retry after attempt consecutive failures.
// The delay grows exponentially from base (attempt 1 == base, attempt 2 == 2*base, ...),
// is capped at max, and then has "equal jitter" applied: the returned value lies in
// [d/2, d) where d is the capped exponential delay. jitter must be in [0.0,1.0).
func nextRetryDelay(base, maxDelay time.Duration, attempt int, jitter float64) time.Duration {
	if base <= 0 {
		base = config.DefaultWriteRetryDelay
	}
	if attempt < 1 {
		attempt = 1
	}

	d := base
	for i := 1; i < attempt; i++ {
		if maxDelay > 0 && d >= maxDelay {
			d = maxDelay
			break
		}
		d *= 2
		if d <= 0 { // overflow, treat as unbounded and clamp below
			d = maxDelay
			break
		}
	}
	if maxDelay > 0 && d > maxDelay {
		d = maxDelay
	}
	if d <= 0 {
		d = base
	}

	// equal jitter: keep half the delay fixed, randomise the other half.
	half := d / 2
	return half + time.Duration(jitter*float64(half))
}

func (a *Auto) setTestHelperFuncs() {
	if a.now == nil {
		a.now = time.Now
	}
	if a.newTimer == nil {
		a.newTimer = func(duration time.Duration) (<-chan time.Time, func() bool) {
			timer := time.NewTimer(duration)
			return timer.C, timer.Stop
		}
	}
	if a.clientActions == nil {
		a.clientActions = ClientActions
	}
	if a.processDone == nil {
		a.processDone = func(readState *ReadState, writeState *WriteState, ttl time.Duration, err error) {
		}
	}
	if a.randFloat == nil {
		a.randFloat = rand.Float64
	}
}
