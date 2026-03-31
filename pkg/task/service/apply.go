package service

import (
	"context"
	"errors"
	"math"
	"time"
)

// MonoApply wraps apply to ensure that consecutive calls to the returned ApplyFunc will cancel the context passed to apply.
func MonoApply[C any](apply ApplyFunc[C]) ApplyFunc[C] {
	var lastCtx context.Context
	var stopLast context.CancelFunc
	return func(ctx context.Context, config C) error {
		if stopLast != nil {
			stopLast()
		}
		lastCtx, stopLast = context.WithCancel(ctx)
		return apply(lastCtx, config)
	}
}

// RetryApply wraps apply, recalling it until it does not return an error or the context is canceled.
// Backoff will be applied between calls to apply.
func RetryApply[C any](apply ApplyFunc[C], opts ...RetryOption) ApplyFunc[C] {
	retry := defaultRetryOptions
	for _, opt := range opts {
		opt(&retry)
	}
	return func(ctx context.Context, config C) error {
		retryCtx := RetryContext{
			T0: time.Now(),
		}
		for {
			attemptCtx, cleanup := context.WithCancel(ctx)
			retryCtx.Err = apply(attemptCtx, config)
			if retryCtx.Err == nil {
				go func() { // cleanup when ctx is no longer needed
					<-ctx.Done()
					cleanup()
				}()

				retryCtx.Delay = 0 // no retry
				retry.Logger(retryCtx)

				return nil
			}
			cleanup()

			// should we abort?
			var abort abortRetry
			if errors.As(retryCtx.Err, &abort) {
				retryCtx.Delay = 0 // no retry
				retry.Logger(retryCtx)
				return abort.err
			}

			// retry logic
			retryCtx.Delay = min(time.Duration(float64(retry.InitialDelay)*math.Pow(retry.Factor, float64(retryCtx.Attempt))), retry.MaxDelay)
			retryCtx.Attempt++
			if retry.MaxAttempts > 0 && retryCtx.Attempt >= retry.MaxAttempts {
				retryCtx.Delay = 0 // no retry
				retry.Logger(retryCtx)
				return retryCtx.Err
			}

			retry.Logger(retryCtx)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryCtx.Delay):
			}
		}
	}
}
