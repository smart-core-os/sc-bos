package history

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

type collector func(ctx context.Context, source config.Source, payloads chan<- []byte)
type getter func(context.Context, chan<- []byte) error

func collectChanges(ctx context.Context, cfg config.Source, pullFn, pollFn getter, changes chan<- []byte, logger *zap.Logger) error {
	if cfg.PollingSchedule != nil {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Until(cfg.PollingSchedule.Schedule.Next(time.Now()))):
				err := pollFn(ctx, changes)
				if err != nil {
					logger.Warn("poll aborted", zap.Error(err))
				}
			}
		}
	}

	err := pull.Changes(ctx, pull.NewFetcher[[]byte](pullFn, pollFn), changes, pull.WithLogger(logger))
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}
