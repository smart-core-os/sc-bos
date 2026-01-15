package bms

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/bms/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

// Patcher represents a single patch that adjusts ReadState.
type Patcher interface {
	Patch(s *ReadState)
}

type PatchFunc func(s *ReadState)

func (p PatchFunc) Patch(s *ReadState) {
	p(s)
}

// setupPatchers configures the automation to pull events from the configured devices and emit patches into changes.
// Any configuration changes, via Configure, should be sent via configChanged chan.
//
// Blocks until ctx is done.
func (a *Auto) setupPatchers(ctx context.Context, configChanged <-chan config.Root, changes chan<- Patcher) error {
	conn := a.clients.ClientConn()
	// Setup the sources that we can pull patches from.
	sources := []*source{
		{
			names: func(cfg config.Root) []string { return cfg.AutoThermostats },
			new: func(name string, logger *zap.Logger) subscriber {
				return &AirTemperaturePatches{
					name:   name,
					client: traits.NewAirTemperatureApiClient(conn),
					logger: logger.Named("airTemperature"),
				}
			},
		},
		{
			names: func(cfg config.Root) (names []string) {
				if cfg.ModeSource.Name == "" {
					return names
				}
				return []string{cfg.ModeSource.Name}
			},
			new: func(name string, logger *zap.Logger) subscriber {
				return &ModePatches{
					name:   name,
					client: traits.NewModeApiClient(conn),
					logger: logger.Named("mode"),
				}
			},
		},
		{
			names: func(cfg config.Root) []string { return cfg.OccupancySensors },
			new: func(name string, logger *zap.Logger) subscriber {
				return &OccupancySensorPatches{
					name:   name,
					client: traits.NewOccupancySensorApiClient(conn),
					logger: logger.Named("occupancy"),
				}
			},
		},
		{
			names: func(cfg config.Root) []string {
				var names []string
				if cfg.AutoModeOATemp != "" {
					names = append(names, cfg.AutoModeOATemp)
				}
				return names
			},
			new: func(name string, logger *zap.Logger) subscriber {
				return &MeanOATempPatches{
					name:          name,
					apiClient:     traits.NewAirTemperatureApiClient(conn),
					historyClient: airtemperaturepb.NewAirTemperatureHistoryClient(conn),
					logger:        logger.Named("meanOATemp"),
				}
			},
		},
	}

	// cancel everything if we're returning.
	defer func() {
		for _, source := range sources {
			for _, cancelFunc := range source.runningSources {
				cancelFunc()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-configChanged:
			if sc := a.processConfig(ctx, e, sources, changes); sc == 0 {
				a.logger.Debug("no sources configured, automation will do nothing")
			}
		}
	}
}

type subscriber interface {
	Subscribe(ctx context.Context, changes chan<- Patcher) error
}

type source struct {
	new   func(name string, logger *zap.Logger) subscriber
	names func(cfg config.Root) []string
	// runningSources, keyed by device name, tracks which sources are currently running.
	// The value can be called to cancel the context used to start that source.
	runningSources map[string]context.CancelFunc
}

func (a *Auto) processConfig(ctx context.Context, cfg config.Root, sources []*source, changes chan<- Patcher) (sourceCount int) {
	logger := a.logger.With(zap.String("auto", cfg.Name))
	for _, source := range sources {
		names := source.names(cfg)
		if source.runningSources == nil && len(names) > 0 {
			source.runningSources = make(map[string]context.CancelFunc, len(names))
		}
		sourcesToStop := shallowCopyMap(source.runningSources)
		for _, name := range names {
			sourceCount++
			logger := logger.With(zap.String("source", name))

			// are we already watching this name?
			if _, ok := sourcesToStop[name]; ok {
				delete(sourcesToStop, name)
				continue
			}
			// I guess not, lets start watching
			ctx, stop := context.WithCancel(ctx)
			source.runningSources[name] = stop
			impl := source.new(name, logger)
			go func() {
				err := impl.Subscribe(ctx, changes)
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				if s, ok := status.FromError(err); ok {
					if s.Code() == codes.Unimplemented {
						logger.Warn(fmt.Sprintf("Subscription does not implement required features: %v", s.Message()))
						return
					}
				}
				if err != nil {
					// todo: handle error, the subscription has failed without us asking it to stop.
					logger.Warn("Subscription ended before it should", zap.Error(err))
				}
			}()
		}

		// stop any sources that are no longer in the config
		for name, cancelFunc := range sourcesToStop {
			cancelFunc()
			delete(source.runningSources, name)
		}
	}

	// update the config in the ReadState too
	changes <- PatchFunc(func(s *ReadState) {
		s.Config = cfg
	})

	return sourceCount
}

func shallowCopyMap[K comparable, V any](m map[K]V) map[K]V {
	n := make(map[K]V, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}

func retryForeverT[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var t T
	err := task.Run(ctx, func(ctx context.Context) (task.Next, error) {
		var err error
		t, err = fn(ctx)
		return 0, err
	}, task.WithBackoff(time.Second, time.Minute), task.WithRetry(task.RetryUnlimited))
	return t, err
}
