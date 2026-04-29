package run

import (
	"context"
	"sync"
)

const DefaultConcurrency = 30

// InParallel runs fns in parallel with a maximum concurrency of concurrency.
// The ctx parameter can be used to cancel the start of new fns.
// Cancellation of running fns is the responsibility of the caller.
// InParallel returns once all started fns have finished or ctx is done.
func InParallel(ctx context.Context, concurrency int, fns ...func()) error {
	jobs := make(chan func(), concurrency)
	done := make(chan struct{}, concurrency)

	// setup workers
	var running sync.WaitGroup
	running.Add(len(fns))
	for range concurrency {
		go func() {
			defer func() {
				done <- struct{}{}
			}()
			for job := range jobs {
				job()
			}
		}()
	}

	// submit jobs
loop:
	for _, fn := range fns {
		select {
		case <-ctx.Done():
			break loop
		case jobs <- fn:
		}
	}
	close(jobs) // no more jobs to submit

	// wait for workers to finish
	for range concurrency {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Collect runs fns in parallel with a maximum concurrency of concurrency.
// The response of running fns[i] is placed into returns[i] or errs[i].
// If ctx is done before all fns have finished, the ctx err is placed into errs[i] for each incomplete fn.
func Collect[T any](ctx context.Context, concurrency int, fns ...func() (T, error)) (returns []T, errs []error) {
	done := make([]bool, len(fns))
	returns, errs = make([]T, len(fns)), make([]error, len(fns))
	jobs := make([]func(), len(fns))
	for i, fn := range fns {
		jobs[i] = func() {
			returns[i], errs[i] = fn()
			done[i] = true
		}
	}
	err := InParallel(ctx, concurrency, jobs...)
	for i, ok := range done {
		if !ok {
			errs[i] = err
		}
	}
	return returns, errs
}
