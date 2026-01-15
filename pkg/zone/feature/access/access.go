package access

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen_accesspb "github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/access/config"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("access")
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

	if len(cfg.AccessPoints) > 0 {
		group := &Group{
			client: gen_accesspb.NewAccessApiClient(f.clients.ClientConn()),
			names:  cfg.AccessPoints,
			logger: logger,
		}

		f.devices.Add(cfg.AccessPoints...)
		announce.Announce(cfg.Name, node.HasTrait(accesspb.TraitName, node.WithClients(gen_accesspb.WrapApi(group))))
	}

	return nil
}
