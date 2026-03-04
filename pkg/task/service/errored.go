package service

import (
	"context"
	"sync"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

// retryLifecycle is used when the factory fails to create a real service.
// It holds an error state and retries the factory when Configure is called, allowing operators
// to fix configuration errors at runtime without restarting the application.
// Once the factory succeeds, all subsequent calls are delegated to the newly created Lifecycle,
// and background state changes are forwarded through our own bus so that existing listeners
// (e.g. GetAndListenState) continue to receive updates without resubscribing.
type retryLifecycle struct {
	mu        sync.Mutex
	state     State
	bus       *minibus.Bus[State]
	inner     Lifecycle             // nil until factory succeeds
	createFn  func() (Lifecycle, error) // retried on each Configure until success
	fwdCancel context.CancelFunc   // cancels the inner-state forwarding goroutine
}

func newRetryLifecycle(err error, createFn func() (Lifecycle, error)) *retryLifecycle {
	return &retryLifecycle{
		state: State{
			Err:         err,
			LastErrTime: time.Now(),
		},
		bus:      &minibus.Bus[State]{},
		createFn: createFn,
	}
}

// Configure retries the factory if no inner Lifecycle has been created yet.
// On factory success the new Lifecycle is configured and stored for future delegation.
// On factory failure the error state is updated and returned.
func (r *retryLifecycle) Configure(data []byte) (State, error) {
	r.mu.Lock()

	if r.inner == nil {
		newSvc, err := r.createFn()
		if err != nil {
			r.state.Err = err
			r.state.LastErrTime = time.Now()
			state := r.state
			r.mu.Unlock()
			go r.bus.Send(context.Background(), state)
			return state, err
		}
		r.inner = newSvc
		r.startForwardingLocked(newSvc)
	}
	inner := r.inner
	r.mu.Unlock()

	state, err := inner.Configure(data)
	r.mu.Lock()
	r.state = state
	r.mu.Unlock()
	go r.bus.Send(context.Background(), state)
	return state, err
}

func (r *retryLifecycle) Start() (State, error) {
	r.mu.Lock()
	if r.inner == nil {
		state := r.state
		r.mu.Unlock()
		return state, state.Err
	}
	inner := r.inner
	r.mu.Unlock()

	state, err := inner.Start()
	r.mu.Lock()
	r.state = state
	// restart forwarding if it was cancelled by a prior Stop
	r.startForwardingLocked(inner)
	r.mu.Unlock()
	go r.bus.Send(context.Background(), state)
	return state, err
}

func (r *retryLifecycle) Stop() (State, error) {
	r.mu.Lock()
	inner := r.inner
	if inner == nil {
		state := r.state
		r.mu.Unlock()
		return state, ErrAlreadyStopped
	}
	cancel := r.fwdCancel
	r.fwdCancel = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	state, err := inner.Stop()
	r.mu.Lock()
	r.state = state
	r.mu.Unlock()
	go r.bus.Send(context.Background(), state)
	return state, err
}

func (r *retryLifecycle) State() State {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inner != nil {
		return r.inner.State()
	}
	return r.state
}

func (r *retryLifecycle) StateChanges(ctx context.Context) <-chan State {
	return r.bus.Listen(ctx)
}

func (r *retryLifecycle) StateAndChanges(ctx context.Context) (State, <-chan State) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.state
	if r.inner != nil {
		state = r.inner.State()
	}
	return state, r.bus.Listen(ctx)
}

// startForwardingLocked starts a goroutine that forwards background state changes from inner
// to our bus (e.g. async applyConfig completions). Must be called with r.mu held.
// No-op if a forwarding goroutine is already running.
// The goroutine exits when Stop cancels fwdCancel.
func (r *retryLifecycle) startForwardingLocked(inner Lifecycle) {
	if r.fwdCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.fwdCancel = cancel
	go func() {
		for state := range inner.StateChanges(ctx) {
			r.mu.Lock()
			r.state = state
			r.mu.Unlock()
			go r.bus.Send(context.Background(), state)
		}
	}()
}
