package once

import (
	"context"
	"errors"
	"sync"
)

// RetryError is like sync.Once except if f returns an err Do will try again the next time it's called.
type RetryError struct {
	mu  sync.Mutex
	cur *attempt // nil if needs init, in progress if init-ing, done without error if successfully done

	recoverPanic bool // for testing, when true panics will be not be left to the system to log and fail tests.
}

// attempt is one call of f.
// Each waiter holds the attempt it waited on, so a retry started after a failure can't change the error it sees.
type attempt struct {
	done chan struct{} // closed when f returns
	err  error         // written before done is closed, read only after
}

var (
	ErrPanicSuppressed = errors.New("panic suppressed")
)

// Do calls f if Do has not successfully been called before.
// Do will return before f completes if ctx is done, f will continue to run until it returns.
// Concurrent calls to Do will wait until the first f completes, returning that error.
// Subsequent calls to Do after an error will invoke f as if Do had not been called before.
func (o *RetryError) Do(ctx context.Context, f func() error) error {
	o.mu.Lock()
	if o.cur != nil {
		// some other routine is working on it, or it's done
		a := o.cur
		o.mu.Unlock()
		return a.wait(ctx)
	}

	// we should work on it
	a := &attempt{done: make(chan struct{})}
	o.cur = a
	o.mu.Unlock()

	// we call f in a go routine to allow this call to return when ctx is done, even if f hasn't completed yet.
	go func() {
		var ran bool
		defer func() {
			// by default we don't actually want to recover from panics as this looses the stack,
			// but it's helpful to avoid incorrect test failures
			if o.recoverPanic {
				recover()
			}
			if !ran && a.err == nil {
				// there was a panic
				a.err = ErrPanicSuppressed
			}
			o.mu.Lock()
			if a.err != nil {
				o.cur = nil // try again next time Do is called
			}
			o.mu.Unlock()
			close(a.done) // release any of those waiting for Do to complete
		}()
		a.err = f()
		ran = true // only not set if f() panics
	}()

	return a.wait(ctx)
}

func (a *attempt) wait(ctx context.Context) error {
	select {
	case <-a.done:
		return a.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
