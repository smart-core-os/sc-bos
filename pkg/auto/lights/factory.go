package lights

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

const AutoType = "lights"

var Factory = auto.FactoryFunc(func(services auto.Services) service.Lifecycle {
	logger := services.Logger.Named("lights")
	impl := PirsTurnLightsOn(services.Node, logger)
	return autoToService(impl, logger)
})

func autoToService(impl *BrightnessAutomation, logger *zap.Logger) service.Lifecycle {
	var started atomic.Bool
	return service.New(func(ctx context.Context, config config.Root) error {
		if started.CompareAndSwap(false, true) {
			err := impl.Start(context.Background())
			if err != nil {
				// Returning an error as part of configuration will stop the Service without calling Stop.
				// Clean up state to make sure our started/stopped state is reflected correctly.
				started.Store(false)
				return fmt.Errorf("start %w", err)
			}
		}

		return impl.configure(config)
	}, service.WithParser(config.Read), service.WithOnStop[config.Root](func() {
		err := impl.Stop()
		if err != nil {
			logger.Error("Error stopping", zap.Error(err))
		}
		started.Store(false)
	}))
}
