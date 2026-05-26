package task

import (
	"context"
	"sync"
	"time"
)

// StartFn is called by an Intermittent to start a background operation.
// The provided context.Context init can be used for initialisation - it is simply the context passed to Intermittent.Attach.
// However, it is not suitable for long-running background operations - create a long-running context using
// context.WithCancel(context.Background()).
// If starting the background operations succeeds, then a StartFn should return a stop function and err should be nil.
// The stop function will be called when the background operation needs to stop.
// If starting the background operation fails, then return a non-nil err; the value of the stop function is ignored.
type StartFn = func(init context.Context) (stop StopFn, err error)
type StopFn = func()

// Intermittent manages the lifecycle of a long-running background operation that needs to outlive a single context.
// A context can be attached to the Intermittent by calling Attach, which will start the background operation if it is not
// already running. Once all attached contexts are cancelled, the background operation will be stopped.
//
// Attach is safe for use by multiple go routines.
type Intermittent struct {
	operation StartFn

	m         sync.Mutex // protects the following state
	listeners int
	stop      StopFn
}

func NewIntermittent(operation StartFn) *Intermittent {
	return &Intermittent{operation: operation}
}

// Attach adds ensures that the background task will remain running for at least as long as listener is not done.
// If the background task is not running it will be started.
//
// Returns an error iff the background task is started by this call and when starting it returns an error.
func (t *Intermittent) Attach(listener context.Context) error {
	t.m.Lock()
	defer t.m.Unlock()

	if t.listeners == 0 {
		stop, err := t.operation(listener)
		if err != nil {
			return err
		}

		// convert nil stop functions into no-ops
		if stop == nil {
			stop = func() {}
		}
		t.stop = stop
	}

	t.listeners++

	go func() {
		// wait for the listener to be cancelled
		<-listener.Done()
		t.m.Lock()
		defer t.m.Unlock()

		t.listeners--
		if t.listeners == 0 {
			t.stop()
		}
	}()

	return nil
}

// IdleKeepAlive immediately runs the operation once for runFor, then re-runs it
// every idleFor thereafter, ensuring the operation stays fresh even with no active listeners.
// If listeners are already attached when a run starts, Attach is a no-op increment
// and the running operation is unaffected.
// Runs until ctx is done.
func (t *Intermittent) IdleKeepAlive(ctx context.Context, idleFor, runFor time.Duration) {
	go func() {
		t.runOnce(ctx, runFor)
		timer := time.NewTimer(idleFor)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				t.runOnce(ctx, runFor)
				timer.Reset(idleFor)
			}
		}
	}()
}

func (t *Intermittent) runOnce(ctx context.Context, runFor time.Duration) {
	attachCtx, cancel := context.WithTimeout(ctx, runFor)
	defer cancel()
	_ = t.Attach(attachCtx)
	<-attachCtx.Done()
}
