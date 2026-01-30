package lights

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/backoffutils"
	"github.com/olebedev/emitter"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
)

// BrightnessAutomation implements turning lights on or off based on occupancy readings from PIRs and other devices.
type BrightnessAutomation struct {
	logger  *zap.Logger
	clients node.ClientConner // client conns are not got until Start

	// bus emits "stop" and "config" events triggered by Stop and Configure.
	bus *emitter.Emitter

	makeActions     func(clientConn node.ClientConner) actions                                       // override for testing
	newTimer        func(duration time.Duration) (<-chan time.Time, func() bool)                     // override for testing
	autoStartTime   time.Time                                                                        // override for testing
	processComplete func(ttl time.Duration, err error, readState *ReadState, writeState *WriteState) // override for testing
}

// PirsTurnLightsOn creates an automation that controls light brightness based on PIR occupancy status.
func PirsTurnLightsOn(clients node.ClientConner, logger *zap.Logger) *BrightnessAutomation {
	return &BrightnessAutomation{
		logger:      logger,
		clients:     clients,
		makeActions: newClientActions,
		newTimer: func(duration time.Duration) (<-chan time.Time, func() bool) {
			t := time.NewTimer(duration)
			return t.C, t.Stop
		},
	}
}

// Start implements Starter and initialises this automation.
// Start may be called before or after Configure.
func (b *BrightnessAutomation) Start(_ context.Context) error {
	if b.bus != nil {
		b.bus.Off("*")
	} else {
		b.bus = emitter.New(1)
	}
	// We make the actions impl here so that we can create the automation before clients are available,
	// so long as they're available before Start is called.
	actions := b.makeActions(b.clients)

	ctx, stop := context.WithCancel(context.Background())
	group, ctx := errgroup.WithContext(ctx)

	// make sure we stop the group when Stop is called
	stopCalled := b.bus.On("stop")
	group.Go(func() error {
		defer b.bus.Off("stop", stopCalled)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-stopCalled:
				stop()
				return context.Canceled
			}
		}
	})

	changes := make(chan Patcher, 10)

	// "actions" the subscriptions. This causes any Pull or Poll or other even sources to start running and placing
	// state changes into the changes chan.
	configChanged := b.bus.On("config")
	group.Go(func() error {
		defer b.bus.Off("config", configChanged)
		err := b.setupReadSources(ctx, configChanged, changes)
		if err != nil {
			return fmt.Errorf("setupReadSources: %w", err)
		}
		return nil
	})

	// readStates receives state that should be processed, for example to work out if lights should be turned on.
	// readStates is how state changes are communicates to the func that processes those state changes.
	readStates := make(chan *ReadState)
	startTime := time.Now()
	if !b.autoStartTime.IsZero() {
		startTime = b.autoStartTime
	}
	initialState := NewReadState(startTime)

	// collect and collate the state changes
	group.Go(func() error {
		err := readStateChanges(ctx, initialState, changes, readStates)
		if err != nil {
			return fmt.Errorf("readStateChanges: %w", err)
		}
		return nil
	})

	// process state changes and execute RPC calls to update the real world
	group.Go(func() error {
		err := b.processStateChanges(ctx, readStates, actions)
		if err != nil {
			return fmt.Errorf("processStateChanges: %w", err)
		}
		return nil
	})

	// log the error when the group stops
	go func() {
		err := group.Wait()
		if errors.Is(err, context.Canceled) {
			return // don't bother logging these as they are expected when stopped
		}
		if err != nil {
			b.logger.Error("automation background tasks stopped", zap.Error(err))
		}
	}()

	// note: we don't wait for the group to complete!
	return nil
}

// Configure updates the configured devices and settings this automation uses.
func (b *BrightnessAutomation) Configure(configData []byte) error {
	cfg, err := config.Read(configData)
	if err != nil {
		return err
	}
	return b.configure(cfg)
}

func (b *BrightnessAutomation) configure(cfg config.Root) error {
	<-b.bus.Emit("config", cfg) // wait for anyone who is listening to apply that config
	return nil
}

// Stop stops the automation from running.
// You must call Start to have automated action occur again.
func (b *BrightnessAutomation) Stop() error {
	b.bus.Emit("stop") // don't wait
	return nil
}

// processStateChanges reads ReadState from a channel and analyses each entry deciding if light levels should be changed.
// This function backs off to processState which has the actual logic for what to do, given a certain state,
// this function handles the channel management, retry logic, TTL on decisions, and all that type of thing.
func (b *BrightnessAutomation) processStateChanges(ctx context.Context, readStates <-chan *ReadState, actions actions) error {
	// retries can happen for a few reasons:
	// - because we periodically wake up to ensure things are still good
	// - because something went wrong, and we want to retry after a delay
	// - because the logic asked us to retry after some ttl
	retryCounter := 0
	var retry <-chan time.Time                 // like time.Timer.C
	cancelRetry := func() bool { return true } // like time.Timer.Stop

	// writeState is only accessed from this go routine.
	writeState := NewWriteState(time.Now())

	var lastProcessedState *ReadState
	var retryReason string
	processStateFn := func(readState *ReadState, reasons ...string) error {
		cancelRetry()
		retryReason = ""

		if readState.Config.LogTriggers {
			logProcessStart(b.logger, lastProcessedState, readState, reasons...)
			lastProcessedState = readState
		}

		actions := actions
		if readState.Config.DryRun {
			actions = nilActions{}
		}
		if readState.Config.LogWrites {
			actions = newLogActions(actions, b.logger)
		}
		actions, actionCounts := newCountActions(actions)
		actions = newCachedActions(actions, readState.Config.WriteCacheExpiry.Or(config.DefaultWriteCacheExpiry))

		t0 := readState.Now()
		writeState.Before()
		for _, reason := range reasons {
			writeState.AddReason(reason) // record why we're running in the completion log
		}
		ttl, err := processState(ctx, readState, writeState, actions)
		writeState.After()
		duration := readState.Now().Sub(t0)

		if err != nil {
			// if the context has been cancelled, stop
			if ctx.Err() != nil {
				return err
			}

			// if the context remains live, schedule another update soon
			retryCounter++
			after := backoffutils.JitterUp(time.Duration(retryCounter)*readState.Config.OnProcessError.BackOffMultiplier.Duration, 0.2)

			if retryCounter > readState.Config.OnProcessError.MaxRetries {
				b.logger.Error("processState failed; too many failures, aborting retires",
					zap.Error(err),
					zap.Int("retryCounter", retryCounter),
				)
				// reset retries to prevent too many repeated attempts
				retryCounter = 0
				if !cancelRetry() {
					<-retry
				}
			} else {
				b.logger.Error("processState failed; scheduling retry",
					zap.Error(err),
					zap.Duration("retryAfter", after),
				)
				retryReason = "error retry"
				ttl = after
			}
		}

		// ensure it's not too long before we wake up, so the lights are refreshed regularly
		// so external changes don't stick around forever
		switch {
		case ttl <= 0:
			retryReason = "refresh"
			writeState.AddReason("refreshEvery:0")
			ttl = readState.Config.RefreshEvery.Duration
		case ttl > readState.Config.RefreshEvery.Duration:
			retryReason = fmt.Sprintf("early refresh:%v->%v", formatDuration(ttl), formatDuration(readState.Config.RefreshEvery.Duration))
			writeState.AddReasonf("refreshEvery:%v->%v", formatDuration(ttl), formatDuration(readState.Config.RefreshEvery.Duration))
			ttl = readState.Config.RefreshEvery.Duration
		case ttl > 0 && retryReason == "":
			retryReason = "ttl"
		}

		// Setup ttl for the transformed model.
		// After this time it should be recalculated.
		retry, cancelRetry = b.newTimer(ttl)

		// log side effects and why they were made
		logProcessComplete(b.logger, readState, writeState, actionCounts, duration, ttl, err)

		if b.processComplete != nil {
			b.processComplete(ttl, err, readState, writeState) // used only for testing
		}

		return nil
	}

	var lastReadState *ReadState

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case readState := <-readStates:
			// reset retries as new valid state received
			if !cancelRetry() {
				<-retry
			}
			retryCounter = 0

			lastReadState = readState
			err := processStateFn(readState)
			if err != nil {
				return err
			}
		case <-retry:
			err := processStateFn(lastReadState, retryReason)
			if err != nil {
				return err
			}
		}
	}
}

type notify interface {
	Emit(topic string, args ...any) chan struct{}
}
