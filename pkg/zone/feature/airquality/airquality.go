package airquality

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/airquality/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airqualitysensorpb"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("airquality")
	f := &feature{
		announcer: node.NewReplaceAnnouncer(services.Node),
		devices:   services.Devices,
		clients:   services.Node,
		logger:    services.Logger,
	}
	f.Service = service.New(service.MonoApply(f.applyConfig))
	return f
})

type feature struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	devices   *zone.Devices
	clients   node.ClientConner
	logger    *zap.Logger
}

func (f *feature) applyConfig(ctx context.Context, cfg config.Root) error {
	announce := f.announcer.Replace(ctx)
	logger := f.logger

	if len(cfg.AirQualitySensors) > 0 {
		group := &Group{
			client: traits.NewAirQualitySensorApiClient(f.clients.ClientConn()),
			names:  cfg.AirQualitySensors,
			logger: logger,
		}

		f.devices.Add(cfg.AirQualitySensors...)
		announce.Announce(cfg.Name, node.HasTrait(trait.AirQualitySensor, node.WithClients(airqualitysensorpb.WrapApi(group))))
	}

	return nil
}
