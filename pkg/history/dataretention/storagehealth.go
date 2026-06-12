package dataretention

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// storageHealth lazily maintains a BoundsCheck reporting disk utilisation for a store.
// The check is created on the first update, so stores without capacity information
// (e.g. SQLite on Windows, or Postgres with no configured max size) never call update
// and so never announce a storage check.
type storageHealth struct {
	checks  *healthpb.Checks
	name    string
	highPct float32
	logger  *zap.Logger

	check *healthpb.BoundsCheck
}

// update reports the current disk utilisation (a percentage, 0-100) to the health check,
// creating the check on first use.
func (h *storageHealth) update(ctx context.Context, utilization float32) {
	if h == nil || h.checks == nil {
		return
	}
	if h.check == nil {
		high := h.highPct
		check, err := h.checks.NewBoundsCheck(h.name, &healthpb.HealthCheck{
			Id:              "storage",
			DisplayName:     "Storage Utilisation",
			Description:     "Reports HIGH when disk utilisation approaches capacity.",
			EquipmentImpact: healthpb.HealthCheck_FUNCTION,
			Check: &healthpb.HealthCheck_Bounds_{
				Bounds: &healthpb.HealthCheck_Bounds{
					Expected: &healthpb.HealthCheck_Bounds_NormalRange{
						NormalRange: &healthpb.HealthCheck_ValueRange{High: healthpb.FloatValue(float64(high))},
					},
					DisplayUnit: "%",
				},
			},
		})
		if err != nil {
			h.logger.Warn("failed to create storage health check", zap.Error(err))
			return
		}
		h.check = check
	}
	h.check.UpdateValue(ctx, healthpb.FloatValue(float64(utilization)))
}

func (h *storageHealth) dispose() {
	if h != nil && h.check != nil {
		h.check.Dispose()
	}
}

// startPolling runs pollLoop in a background goroutine and returns an Undo that stops the
// loop, waits for it to exit, then disposes the health check.
//
// Disposing only after the goroutine has returned is what makes this safe: storageHealth's
// check (a healthpb.BoundsCheck) is not safe for concurrent use, so an in-flight update must
// never overlap dispose. The Undo cancels the loop's context (aborting any in-flight update)
// and blocks on the done channel before calling dispose.
func startPolling(ctx context.Context, model *dataretentionpb.Model, alwaysPoll bool, health *storageHealth, update func(ctx context.Context, full bool)) node.Undo {
	ctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		pollLoop(ctx, model, alwaysPoll, update)
	}()
	return func() {
		stop()
		<-done
		health.dispose()
	}
}

// pollLoop drives update at a fixed interval until ctx is done, calling it once
// immediately so GetDataRetention returns a value without waiting for the first tick.
//
// update's full argument tells it whether to refresh expensive figures (e.g. SQLite row
// counts, which are a full table scan) alongside the cheap ones. full is true at startup,
// whenever a PullDataRetention subscriber connects, and on every poll while at least one
// subscriber is connected; unsubscribed polls pass false so updates that nobody is
// watching stay cheap.
//
// When alwaysPoll is false the loop idles (no polling) until a subscriber connects,
// avoiding work when nobody is watching. When alwaysPoll is true the loop polls
// unconditionally — used when a storage HealthCheck is an always-on consumer of the
// polled data.
func pollLoop(ctx context.Context, model *dataretentionpb.Model, alwaysPoll bool, update func(ctx context.Context, full bool)) {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	update(ctx, true)
	for {
		switch {
		case model.HasSubscribers():
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				update(ctx, true)
			}
		case alwaysPoll:
			select {
			case <-ctx.Done():
				return
			case <-model.WaitForSubscriber():
				update(ctx, true)
				tick.Reset(30 * time.Second)
			case <-tick.C:
				update(ctx, false)
			}
		default:
			select {
			case <-ctx.Done():
				return
			case <-model.WaitForSubscriber():
				update(ctx, true)
				tick.Reset(30 * time.Second)
			}
		}
	}
}
