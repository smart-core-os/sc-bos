package history

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

type collector func(ctx context.Context, source config.Source, payloads chan<- []byte, logger *zap.Logger)
type getter func(context.Context, chan<- []byte) error

// collectChanges blocks until ctx is done, sending changes from source to changes.
// A nil error means ctx is done, any other error means collection stopped early.
func collectChanges(ctx context.Context, cfg config.Source, pullFn, pollFn getter, changes chan<- []byte, logger *zap.Logger) error {
	pullFn = keepCtxErrs(pullFn)
	pollFn = withTimeout(pollFn, cfg.TimeoutDuration())

	if cfg.PollingSchedule != nil {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Until(cfg.PollingSchedule.Schedule.Next(time.Now()))):
				err := pollFn(ctx, changes)
				if err != nil && ctx.Err() == nil {
					logger.Warn("poll aborted", zap.Error(err))
				}
			}
		}
	}

	err := pull.Changes(ctx, pull.NewFetcher[[]byte](pullFn, pollFn), changes, pull.WithLogger(logger))
	if ctx.Err() != nil {
		return nil
	}
	return err
}

// withTimeout bounds each call to poll by timeout.
// Errors are passed through keepCtxErrs, so a poll that times out is retried rather than ending collection.
//
// The timeout covers fetching, not handing payloads to changes.
// Collectors dedupe a value before sending it, so a payload abandoned while storage is slow would never be sent again.
func withTimeout(poll getter, timeout time.Duration) getter {
	poll = keepCtxErrs(poll)
	return func(ctx context.Context, changes chan<- []byte) error {
		pollCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var got [][]byte
		buf := make(chan []byte)
		drained := make(chan struct{})
		go func() {
			defer close(drained)
			for payload := range buf {
				got = append(got, payload)
			}
		}()
		err := poll(pollCtx, buf)
		close(buf)
		<-drained

		if err != nil && ctx.Err() == nil && pollCtx.Err() != nil {
			err = status.Errorf(codes.DeadlineExceeded, "poll timed out after %s: %v", timeout, err)
		}
		for _, payload := range got {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case changes <- payload:
			}
		}
		return err
	}
}

// keepCtxErrs replaces context errors returned by fn with an equivalent status error, unless the ctx passed to fn is done.
//
// pull.Changes stops for good when a getter returns an error matching context.Canceled or context.DeadlineExceeded,
// assuming it means ctx is done.
// That isn't true of a deadline set further down, by withTimeout or the server,
// and stopping then would end collection for this source without anything noticing.
func keepCtxErrs(fn getter) getter {
	return func(ctx context.Context, changes chan<- []byte) error {
		err := fn(ctx, changes)
		if ctx.Err() == nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
			return status.FromContextError(err).Err()
		}
		return err
	}
}
