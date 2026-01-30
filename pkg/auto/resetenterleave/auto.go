// Package resetenterleave provides an auto that resets enter/leave totals based on a cron-like schedule.
// With this automation you can schedule a call to EnterLeave.ResetEnterLeaveTotals at 4am every day, for example.
package resetenterleave

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/resetenterleave/config"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

const AutoName = "resetenterleave"

var Factory auto.Factory = factory{}

type factory struct{}

func (f factory) New(services auto.Services) service.Lifecycle {
	services.Logger = services.Logger.Named(AutoName)
	a := &Auto{
		services: services,
	}
	a.Service = service.New(service.MonoApply(a.applyConfig))
	return a
}

type Auto struct {
	*service.Service[config.Root]
	services auto.Services
}

func (a *Auto) applyConfig(ctx context.Context, cfg config.Root) error {
	if len(cfg.Devices) == 0 {
		return nil
	}

	elClient := traits.NewEnterLeaveSensorApiClient(a.services.Node.ClientConn())

	sched := cfg.Schedule
	if sched == nil {
		sched = jsontypes.MustParseSchedule("0 0 * * *")
	}
	go func() {
		t := time.Now()
		for {
			next := sched.Next(t)
			tick := time.NewTimer(next.Sub(t))
			select {
			case <-ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				grp, ctx := errgroup.WithContext(ctx)
				for _, device := range cfg.Devices {
					device := device
					grp.Go(func() error {
						resetTotals(ctx, elClient, device, a.services.Logger)
						return nil
					})
				}
				err := grp.Wait()
				if err != nil {
					return
				}
				t = time.Now()
			}
		}
	}()
	return nil
}

func resetTotals(ctx context.Context, client traits.EnterLeaveSensorApiClient, name string, logger *zap.Logger) {
	logger = logger.With(zap.String("name", name))
	err := task.Run(ctx, func(ctx context.Context) (task.Next, error) {
		_, err := client.ResetEnterLeaveTotals(ctx, &traits.ResetEnterLeaveTotalsRequest{
			Name: name,
		})
		if status.Code(err) == codes.Unimplemented {
			return 0, nil
		}
		return 0, err
	}, task.WithRetry(5), task.WithBackoff(time.Second, 30*time.Second))
	if err != nil {
		logger.Warn("failed to reset enter/leave totals", zap.Error(err))
	} else {
		logger.Debug("enter/leave totals reset")
	}
}
