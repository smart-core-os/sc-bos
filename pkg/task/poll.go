package task

import (
	"context"
	"fmt"
	"time"
)

// Poll creates a task that calls a function at a regular interval while it is running.
// The action will not be run until the returned Intermittent.Attach is called.
func Poll(action func(context.Context), interval time.Duration) *Intermittent {
	return PollErr(func(ctx context.Context) error {
		action(ctx)
		return nil
	}, WithPollInterval(interval))
}

// PollErrTask is a func passed to PollErr.
type PollErrTask = func(context.Context) error

// PollErr calls action at regular intervals, during error the interval may be different.
// The action will not be run until the returned Intermittent.Attach is called.
//
// The default interval is 1 second.
// The default error backoff is to retry the action again after interval has passed.
func PollErr(action PollErrTask, opts ...PollOption) *Intermittent {
	// apply defaults
	opts = append([]PollOption{WithPollInterval(time.Second)}, opts...)
	p := poller{}
	for _, opt := range opts {
		opt(&p)
	}

	return NewIntermittent(func(_ context.Context) (StopFn, error) {
		// The work context is intentionally derived from context.Background(), not the
		// init context passed here. The poll goroutine's lifetime is controlled by the
		// returned cancel (StopFn), independent of the initialisation caller's context.
		ctx, cancel := context.WithCancel(context.Background())
		go p.poll(ctx, func(ctx context.Context) error {
			return action(ctx)
		})
		return cancel, nil
	})
}

// PollOption allows configuration of the PollErr task.
type PollOption func(*poller)

// WithPollInterval sets the interval between calls to the action.
// If no error backoff is configured the task will be reattempted after this interval too.
func WithPollInterval(interval time.Duration) PollOption {
	return func(p *poller) {
		p.interval = interval
	}
}

// WithPollErrBackoff sets the backoff between retries after an error.
// These timings can be shorter or longer than the success interval.
func WithPollErrBackoff(initial, max time.Duration, scale float64) PollOption {
	if scale != 0 && scale < 1 {
		panic(fmt.Sprintf("invalid scale, should be 0 or >= 1, got %v", scale))
	}
	if initial <= 0 {
		panic(fmt.Sprintf("invalid initial, should be > 0, got %v", initial))
	}
	if max != 0 && max < initial {
		panic(fmt.Sprintf("invalid max, should be 0 or >= initial, got %v", max))
	}
	return func(p *poller) {
		p.errBackoff.initial = initial
		p.errBackoff.max = max
		p.errBackoff.scale = scale
	}
}

// PollAttemptCallback is called after each poll attempt during PollErr.
// If the action returned an error, this will be passed to the callback.
// Returning true from the callback will abort the poll, no more attempts will be made.
type PollAttemptCallback = func(PollState, error) bool

// WithPollAttemptCallback allows a callback to be called after each attempt.
// Returning true from the callback will abort the poll, no more attempts will be made.
func WithPollAttemptCallback(f PollAttemptCallback) PollOption {
	return func(p *poller) {
		p.attemptComplete = f
	}
}

// PollState describes the running state of a PollErr task.
type PollState struct {
	// Totals for how many successful and failed action calls have been made.
	TotalErrors, TotalSuccesses int
	// Counts for how many error or success calls have been made since the last opposite event.
	ErrorsSinceSuccess  int
	SuccessesSinceError int

	NextDelay time.Duration
}

type poller struct {
	interval   time.Duration
	errBackoff struct {
		initial time.Duration
		max     time.Duration
		scale   float64
	}
	attemptComplete PollAttemptCallback
}

func (p poller) poll(ctx context.Context, run func(context.Context) error) {
	state := PollState{NextDelay: p.interval}
	lastDelay := state.NextDelay
	ticker := time.NewTicker(lastDelay)
	defer ticker.Stop()
	for {
		err := run(ctx)
		if err != nil {
			state.ErrorsSinceSuccess++
			state.SuccessesSinceError = 0
			if state.ErrorsSinceSuccess == 1 {
				// first err
				if p.errBackoff.initial > 0 {
					state.NextDelay = p.errBackoff.initial
				}
			} else {
				if p.errBackoff.scale > 1 {
					state.NextDelay = time.Duration(float64(state.NextDelay) * p.errBackoff.scale)
					if p.errBackoff.max > 0 && state.NextDelay > p.errBackoff.max {
						state.NextDelay = p.errBackoff.max
					}
				}
			}
		} else {
			state.TotalSuccesses++
			state.ErrorsSinceSuccess = 0
			state.SuccessesSinceError++
			state.NextDelay = p.interval
		}

		if lastDelay != state.NextDelay {
			lastDelay = state.NextDelay
			ticker.Reset(state.NextDelay)
		}

		if p.attemptComplete != nil && p.attemptComplete(state, err) {
			return // abort
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				// If both <-ctx.Done() and <-ticker.C are ready, <-ticker.C could be selected, which we don't want.
				// If the context is complete, we want to stop the poll immediately.
				return
			}
		}
	}
}
