package merge

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

func startPoll(init context.Context, name string, pollDelay, pollTimeout time.Duration, logger *zap.Logger, pollPeer func(ctx context.Context) error) (task.StopFn, error) {
	runUntil, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(pollDelay)
	go func() {
		cleanup := func() {}
		defer func() { cleanup() }()
		for {
			cleanup()
			ctx, stop := context.WithTimeout(runUntil, pollTimeout)
			cleanup = stop
			err := pollPeer(ctx)
			comm.LogPollError(logger, fmt.Sprintf("%s poll error", name), err)
			select {
			case <-ticker.C:
			case <-runUntil.Done():
				return
			}
		}
	}()
	return cancel, nil
}

// pollUntil calls poll until test returns true.
// Returns early with error if
//
//  1. ctx is done
//  2. pollPeer returns an error
//  3. timeout has passed if the ctx doesn't have a non-zero ctx.Deadline assigned
//     or time.Now() + timeout is earlier than the deadline assigned to ctx
//
// An backoff delay will be added between each call to pollPeer
func pollUntil[T any](ctx context.Context, timeout time.Duration, poll func(context.Context) (T, error), test func(T) bool) (T, error) {
	fail := func(err error) (T, error) {
		var zero T
		return zero, err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var delay time.Duration
	delayMulti := 1.2
	var attempt int
	for {
		attempt++ // start with attempt 1 (not 0)

		res, err := poll(ctx)
		if err != nil {
			return fail(err)
		}

		if test(res) {
			return res, nil
		}

		if delay == 0 {
			delay = 10 * time.Millisecond
		} else {
			delay = time.Duration(float64(delay) * delayMulti)
		}

		select {
		case <-ctx.Done():
			return fail(fmt.Errorf("attempt %d: %w", attempt, ctx.Err()))
		case <-time.After(delay):
		}
	}
}
