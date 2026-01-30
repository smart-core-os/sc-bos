// Package openclose adds the open/close trait to a zone.
//
// Groups of open/close devices have their data merged by averaging out the position of each direction separately.
// Resistance is merged based on the following priority order: UNSPECIFIED < SLOW < REDUCED_MOTION < HELD.
package openclose

import (
	"context"
	"path"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/openclose/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/openclosepb"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("openclose")
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

	apiClient := traits.NewOpenCloseApiClient(f.clients.ClientConn())
	announceGroup := func(name string, devices []string) {
		if len(devices) == 0 {
			return
		}

		group := &Group{
			apiClient: apiClient,
			names:     devices,
			logger:    logger,
		}
		f.devices.Add(devices...)
		announce.Announce(name, node.HasTrait(trait.OpenClose, node.WithClients(openclosepb.WrapApi(group))))
	}

	announceGroup(cfg.Name, cfg.OpenClose)
	for name, openClosers := range cfg.OpenCloseGroups {
		announceGroup(path.Join(cfg.Name, name), openClosers)
	}

	return nil
}
