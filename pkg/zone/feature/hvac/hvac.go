package hvac

import (
	"context"
	"path"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/hvac/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("hvac")
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
	client := traits.NewAirTemperatureApiClient(f.clients.ClientConn())
	publish := func(name string, t config.Thermostat) error {
		group := &Group{
			client:   client,
			names:    t.Thermostats,
			readOnly: t.ReadOnlyThermostat,
			logger:   logger,
		}
		f.devices.Add(t.Thermostats...)
		announce.Announce(name, node.HasTrait(trait.AirTemperature, node.WithClients(airtemperaturepb.WrapApi(group))))
		return nil
	}

	if len(cfg.Thermostats) > 0 {
		if err := publish(cfg.Name, cfg.Thermostat); err != nil {
			return err
		}
	}

	for k, t := range cfg.ThermostatGroups {
		name := path.Join(cfg.Name, k)
		if err := publish(name, t); err != nil {
			return err
		}
	}

	return nil
}
