package status

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/status/config"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("status")
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

	if len(cfg.StatusLogs) > 0 || cfg.StatusLogAll {
		client := gen_statuspb.NewStatusApiClient(f.clients.ClientConn())

		f.devices.Add(cfg.StatusLogs...)
		if cfg.StatusLogAll {
			go func() {
				select {
				case <-ctx.Done():
					return
				case <-f.devices.Frozen():
					names := f.devices.Names()
					if len(names) == 0 {
						logger.Warn("zone has no devices that implement status")
						return
					}
					logger.Debug("zone discovered status devices", zap.Strings("names", names))
					group := &Group{
						client: client,
						names:  names,
						logger: logger,
					}
					announce.Announce(cfg.Name, node.HasTrait(statuspb.TraitName, node.WithClients(gen_statuspb.WrapApi(group))))
				}
			}()
		} else {
			group := &Group{
				client: client,
				names:  cfg.StatusLogs,
				logger: logger,
			}
			announce.Announce(cfg.Name, node.HasTrait(statuspb.TraitName, node.WithClients(gen_statuspb.WrapApi(group))))
		}
	}

	return nil
}
