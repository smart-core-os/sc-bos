package lights

import (
	"context"
	"errors"
	"fmt"

	"github.com/olebedev/emitter"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// Patcher represents a single patch that adjusts ReadState.
type Patcher interface {
	Patch(s *ReadState)
}

type PatchFunc func(s *ReadState)

func (p PatchFunc) Patch(s *ReadState) {
	p(s)
}

type subscriber interface {
	Subscribe(ctx context.Context, changes chan<- Patcher) error
}

// setupReadSources configures the automation to pull events from the configured devices and emit patches into changes.
// Any configuration changes, via Configure, should be sent via configChanged chan which matches the return type for Emitter.On.
//
// Blocks until fatal errors in the subscriptions or ctx is done.
func (b *BrightnessAutomation) setupReadSources(ctx context.Context, configChanged <-chan emitter.Event, changes chan<- Patcher) error {
	conn := b.clients.ClientConn()
	// Setup the sources that we can pull patches from.
	sources := []*source{
		{
			names: func(cfg config.Root) []deviceName { return cfg.OccupancySensors },
			new: func(name deviceName, logger *zap.Logger) subscriber {
				return &OccupancySensorPatches{name: name, client: traits.NewOccupancySensorApiClient(conn), logger: logger}
			},
		},
		{
			names: func(cfg config.Root) []deviceName { return cfg.BrightnessSensors },
			new: func(name deviceName, logger *zap.Logger) subscriber {
				return &BrightnessSensorPatches{name: name, client: traits.NewBrightnessSensorApiClient(conn), logger: logger}
			},
		},
		{
			names: func(cfg config.Root) (names []deviceName) {
				return cfg.OnButtons
			},
			new: func(name deviceName, logger *zap.Logger) subscriber {
				return &ButtonPatches{
					name:   name,
					client: buttonpb.NewButtonApiClient(conn),
					logger: logger,
				}
			},
		},
		{
			names: func(cfg config.Root) (names []deviceName) {
				return cfg.OffButtons
			},
			new: func(name deviceName, logger *zap.Logger) subscriber {
				return &ButtonPatches{
					name:   name,
					client: buttonpb.NewButtonApiClient(conn),
					logger: logger,
				}
			},
		},
		{
			names: func(cfg config.Root) (names []deviceName) {
				return cfg.ToggleButtons
			},
			new: func(name deviceName, logger *zap.Logger) subscriber {
				return &ButtonPatches{
					name:   name,
					client: buttonpb.NewButtonApiClient(conn),
					logger: logger,
				}
			},
		},
		{
			names: func(cfg config.Root) (names []deviceName) {
				if cfg.ModeSource == "" {
					return names
				}
				return []deviceName{cfg.ModeSource}
			},
			new: func(name deviceName, logger *zap.Logger) subscriber {
				return &ModePatches{
					name:   name,
					client: traits.NewModeApiClient(conn),
					logger: logger,
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
			cfg := e.Args[0].(config.Root)
			// update the config in the ReadState too
			changes <- PatchFunc(func(s *ReadState) {
				s.Config = cfg
			})
			if sc := b.processConfig(ctx, cfg, sources, changes); sc == 0 {
				b.logger.Debug("no sources configured, automation will do nothing")
			}
		}
	}
}

type source struct {
	new   func(name deviceName, logger *zap.Logger) subscriber
	names func(cfg config.Root) []deviceName
	// runningSources, keyed by device name, tracks which sources are currently running.
	// The value can be called to cancel the context used to start that source.
	runningSources map[deviceName]context.CancelFunc
}

func (b *BrightnessAutomation) processConfig(ctx context.Context, cfg config.Root, sources []*source, changes chan<- Patcher) (sourceCount int) {
	if cfg.OnProcessError == nil {
		cfg.OnProcessError = &config.OnProcessError{}
	}
	if cfg.OnProcessError.BackOffMultiplier == nil || cfg.OnProcessError.BackOffMultiplier.Duration.Nanoseconds() <= 0 {
		cfg.OnProcessError.BackOffMultiplier = &jsontypes.Duration{Duration: config.DefaultBackOffMultiplier}
	}
	if cfg.OnProcessError.MaxRetries < 0 {
		cfg.OnProcessError.MaxRetries = config.DefaultMaxRetries
	}
	if cfg.RefreshEvery == nil || cfg.RefreshEvery.Duration.Nanoseconds() <= 0 {
		cfg.RefreshEvery = &jsontypes.Duration{Duration: config.DefaultRefreshEvery}
	}

	for _, source := range sources {
		names := source.names(cfg)
		if source.runningSources == nil && len(names) > 0 {
			source.runningSources = make(map[deviceName]context.CancelFunc, len(names))
		}
		sourcesToStop := shallowCopyMap(source.runningSources)
		for _, name := range names {
			sourceCount++
			logger := b.logger.With(zap.String("source", name))

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

	return sourceCount
}

func shallowCopyMap[K comparable, V any](m map[K]V) map[K]V {
	n := make(map[K]V, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}
