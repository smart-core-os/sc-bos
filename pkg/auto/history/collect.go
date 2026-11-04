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

type collector func(ctx context.Context, source config.Source, payloads chan<- []byte, rec *recency, logger *zap.Logger)
type getter func(context.Context, chan<- []byte) error

// heartbeatRetryDelay is how long to wait before asking again when a heartbeat could not read the source,
// or its reading could not be stored.
// Short relative to any sensible maxInterval, so a missed beat costs minutes of history rather than an interval of it.
const heartbeatRetryDelay = time.Minute

// maxHeartbeatRetries bounds the attempts spent on one missed beat,
// so a source that stays unreadable is asked a few times per maxInterval rather than every minute.
const maxHeartbeatRetries = 3

// collectChanges blocks until ctx is done, sending changes from source to changes.
// A nil error means ctx is done, any other error means collection stopped early.
//
// When rec has a maxInterval, a heartbeat also polls the source whenever nothing has been accepted for that long.
func collectChanges(ctx context.Context, cfg config.Source, pullFn, pollFn getter, changes chan<- []byte, rec *recency, logger *zap.Logger) error {
	pullFn = keepCtxErrs(pullFn)
	pollFn = withTimeout(withReadStart(pollFn), cfg.TimeoutDuration())

	if rec.maxInterval > 0 {
		ctx, cancel := context.WithCancel(ctx)
		beating := make(chan struct{})
		go func() {
			defer close(beating)
			runHeartbeat(ctx, rec, pollFn, changes, logger)
		}()
		// callers may stop reading changes once we return
		defer func() {
			cancel()
			<-beating
		}()
	}

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

// runHeartbeat polls the source each time rec has gone maxInterval without accepting a reading, until ctx is done.
// The poll's deduper accepts the reading because of its age, so an unchanged value is still recorded.
//
// It must be a live read of the source, never a replay of a value we already hold:
// a failed read records nothing, which is what lets a gap in history mean the source couldn't be read.
func runHeartbeat(ctx context.Context, rec *recency, poll getter, changes chan<- []byte, logger *zap.Logger) {
	var (
		// retryAt, when set, is when to retry a beat that couldn't read the source or whose reading couldn't be stored.
		retryAt time.Time
		retries int
		// notBefore stops a source whose reads keep failing, leaving rec.next() in the past, from spinning the loop.
		notBefore time.Time
	)
	retry := func() bool {
		if retries >= maxHeartbeatRetries {
			retries = 0
			retryAt = time.Time{}
			notBefore = time.Now().Add(rec.maxInterval)
			return false
		}
		retries++
		retryAt = time.Now().Add(min(heartbeatRetryDelay, rec.maxInterval))
		return true
	}

	for {
		at := retryAt
		if at.IsZero() {
			at = rec.next()
			if at.Before(notBefore) {
				at = notBefore
			}
		}
		timer := time.NewTimer(time.Until(at))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-rec.wake:
			timer.Stop()
			logger.Debug("heartbeat retrying after a failed write", zap.Bool("retrying", retry()))
			continue
		case <-timer.C:
		}

		if retryAt.IsZero() {
			now := time.Now()
			if now.Before(rec.next()) {
				continue // a reading was accepted while we waited
			}
			retries = 0
			notBefore = now.Add(rec.maxInterval)
		}
		retryAt = time.Time{}

		err := poll(ctx, changes)
		switch {
		case err == nil:
		case ctx.Err() != nil:
			return
		case status.Code(err) == codes.Unimplemented:
			logger.Warn("source can't be read on demand, maxInterval ignored", zap.Error(err))
			return
		case status.Code(err) == codes.Unavailable:
			logger.Debug("heartbeat could not read source", zap.Error(err), zap.Bool("retrying", retry()))
		default:
			logger.Warn("heartbeat could not read source", zap.Error(err), zap.Bool("retrying", retry()))
		}
	}
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
