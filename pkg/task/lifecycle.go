package task

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/util/state"
)

// Starter describes types that can be started.
type Starter interface {
	// Start instructs the automation to start.
	// The ctx represents how long the type can spend starting before it should give up.
	Start(ctx context.Context) error
}

// Stopper describes types that can be stopped.
type Stopper interface {
	// Stop instructs the automation to stop what it is doing.
	Stop() error
}

// ErrCannotBeStopped is returned when attempting to Stop an automation that does not implement Stopper.
var ErrCannotBeStopped = errors.New("cannot be stopped")

// Stop attempts to stop the given automation.
// If the automation does not implement Stopper then ErrCannotBeStopped will be returned.
func Stop(s any) error {
	if impl, ok := s.(Stopper); ok {
		return impl.Stop()
	}
	return ErrCannotBeStopped
}

// Stoppable returns whether the given automation can be stopped.
func Stoppable(s any) bool {
	_, ok := s.(Stopper)
	return ok
}

// Configurer describes types that can be configured.
// Configuration data is represented as a []byte and the automation is expected to decode this data into an internally
// significant data structure.
type Configurer interface {
	Configure(configData []byte) error
}

// ErrCannotBeConfigured is returned when attempting to Configure an automation that does not implement Configurer.
var ErrCannotBeConfigured = errors.New("cannot be configured")

// Configure attempts to configure the given automation.
// If the automation does not implement Configurer then ErrCannotBeConfigured will be returned.
func Configure(s any, configData []byte) error {
	if impl, ok := s.(Configurer); ok {
		return impl.Configure(configData)
	}
	return ErrCannotBeConfigured
}

// Configurable returns whether the given automation can be configured.
func Configurable(s any) bool {
	_, ok := s.(Configurer)
	return ok
}

// RunConfigFunc is called when a drivers config changes and should apply those changes.
// The ctx will be cancelled if the driver stops or the config is replaced.
// RunConfigFunc should only block while cfg is being applied, any long running tasks should run in separate go routines.
type RunConfigFunc[C any] func(ctx context.Context, cfg C) error

// Lifecycle manages the lifecycle of a driver as per task.Starter, task.Configurer, and task.Stopper.
// Embed Lifecycle into your own driver type and provide a RunConfigFunc that binds your driver based on config.
type Lifecycle[C any] struct {
	// ApplyConfig is a function that gets called each time config is changed.
	ApplyConfig RunConfigFunc[C]
	Logger      *zap.Logger
	// ReadConfig converts bytes into C.
	// Defaults to json.Unmarshal.
	ReadConfig func(bytes []byte) (C, error)

	status *state.Manager[Status] // allows us to implement Stateful

	// these track our state and config changes
	configC chan C
	stopCtx context.Context
	stop    context.CancelFunc
}

// NewLifecycle creates a new Lifecycle that calls runFunc each time config is loaded.
func NewLifecycle[C any](runFunc RunConfigFunc[C]) *Lifecycle[C] {
	return &Lifecycle[C]{
		Logger:      zap.NewNop(),
		ApplyConfig: runFunc,
		ReadConfig: func(bytes []byte) (C, error) {
			var c C
			err := json.Unmarshal(bytes, &c)
			return c, err
		},
		status: state.NewManager(StatusInactive),
	}
}

// Start makes this driver available to be configured.
// Call Stop when you're done with the driver to free up resources.
//
// Start must be called before Configure.
// Once started Configure and Stop may be called from any go routine.
func (s *Lifecycle[C]) Start(_ context.Context) error {
	// We implement a main loop pattern to avoid locks.
	// Methods like Configure and Stop both push messages onto channels
	// which we select on in a loop using a single go routine, avoiding any
	// locking issues (that we have to deal with ourselves).

	s.configC = make(chan C, 5)
	s.stopCtx, s.stop = context.WithCancel(context.Background())
	go func() {
		s.status.Update(StatusActive)
		defer s.status.Update(StatusInactive)

		// allow stopping and re-running based on new config
		var cfgCtx context.Context
		cfgStop := func() {}
		defer func() {
			cfgStop()
		}()

		for {
			select {
			case cfg := <-s.configC:
				s.status.Update(StatusLoading)

				cfgStop()
				// It's called in the defer
				//goland:noinspection GoVetLostCancel
				cfgCtx, cfgStop = context.WithCancel(s.stopCtx)

				err := s.ApplyConfig(cfgCtx, cfg)
				if err != nil {
					s.status.Update(StatusError)
					s.Logger.Error("failed to apply config update", zap.Error(err))
					continue
				}
				s.status.Update(StatusActive)
			case <-s.stopCtx.Done():
				//goland:noinspection GoVetLostCancel
				return
			}
		}
	}()

	return nil
}

// Configure instructs the driver to setup and announce any devices found in configData.
// configData should be an encoded JSON object matching config.Root.
//
// Configure must not be called before Start, but once Started can be called concurrently.
func (s *Lifecycle[C]) Configure(configData []byte) error {
	if s.configC == nil {
		return errors.New("not started")
	}
	c, err := s.ReadConfig(configData)
	if err != nil {
		return err
	}
	s.configC <- c
	return nil
}

// Stop stops the driver and releases resources.
// Stop races with Start before Start has completed, but can be called concurrently once started.
func (s *Lifecycle[C]) Stop() error {
	if s.stop == nil {
		// not started
		return nil
	}
	s.stop()
	return nil
}

func (s *Lifecycle[C]) WaitForStateChange(ctx context.Context, sourceState Status) error {
	return s.status.WaitForStateChange(ctx, sourceState)
}

func (s *Lifecycle[C]) CurrentState() Status {
	return s.status.CurrentState()
}
