package electric

import (
	"context"
	"path"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/electric/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/electricpb"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("electric")
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
	if len(cfg.Electrics) == 0 && len(cfg.ElectricGroups) == 0 {
		return nil
	}
	client := traits.NewElectricApiClient(f.clients.ClientConn())

	announce := f.announcer.Replace(ctx)
	logger := f.logger

	announceGroup := func(name string, devices []string) {
		if len(devices) == 0 {
			return
		}
		group := &Group{
			client: client,
			names:  devices,
			logger: logger,
		}
		f.devices.Add(devices...)
		announce.Announce(name, node.HasTrait(trait.Electric, node.WithClients(electricpb.WrapApi(group))))
	}

	announceGroup(cfg.Name, cfg.Electrics)
	for name, group := range cfg.ElectricGroups {
		announceGroup(path.Join(cfg.Name, name), group)
	}

	return nil
}
