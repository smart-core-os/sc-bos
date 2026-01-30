package mode

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/mode/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/modepb"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("mode")
	f := &feature{
		announcer: node.NewReplaceAnnouncer(services.Node),
		devices:   services.Devices,
		clients:   services.Node,
		logger:    services.Logger,
	}
	f.Service = service.New(service.MonoApply(f.applyConfig), service.WithParser(config.ReadConfigBytes))
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
	if len(cfg.Modes) == 0 {
		return nil
	}
	announce := f.announcer.Replace(ctx)
	logger := f.logger

	f.devices.Add(cfg.AllDeviceNames()...)
	group := &Group{
		client: traits.NewModeApiClient(f.clients.ClientConn()),
		cfg:    cfg,
		logger: logger,
	}
	announce.Announce(cfg.Name, node.HasTrait(trait.Mode, node.WithClients(modepb.WrapApi(group), modepb.WrapInfo(group))))

	return nil
}
