package service

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

// erroredLifecycle is a permanent-error Lifecycle used when the factory fails to create a real service.
// It holds a fixed error state and never starts or accepts config.
type erroredLifecycle struct {
	state State
	bus   *minibus.Bus[State]
}

func newErroredLifecycle(err error) *erroredLifecycle {
	return &erroredLifecycle{
		state: State{
			Err:         err,
			LastErrTime: time.Now(),
		},
		bus: &minibus.Bus[State]{},
	}
}

func (s *erroredLifecycle) Start() (State, error) {
	return s.state, s.state.Err
}

func (s *erroredLifecycle) Configure([]byte) (State, error) {
	return s.state, s.state.Err
}

func (s *erroredLifecycle) Stop() (State, error) {
	return s.state, ErrAlreadyStopped
}

func (s *erroredLifecycle) State() State {
	return s.state
}

func (s *erroredLifecycle) StateChanges(ctx context.Context) <-chan State {
	return s.bus.Listen(ctx)
}

func (s *erroredLifecycle) StateAndChanges(ctx context.Context) (State, <-chan State) {
	return s.state, s.bus.Listen(ctx)
}
