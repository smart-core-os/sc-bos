package soundsensor

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/soundsensor/config"
)

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("soundsensor")
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

	if len(cfg.SoundSensors) > 0 {
		group := &Group{
			client: soundsensorpb.NewSoundSensorApiClient(f.clients.ClientConn()),
			names:  cfg.SoundSensors,
			logger: logger,
		}

		f.devices.Add(cfg.SoundSensors...)
		announce.Announce(cfg.Name,
			node.HasServer(soundsensorpb.RegisterSoundSensorApiServer, soundsensorpb.SoundSensorApiServer(group)),
			node.HasTrait(soundsensorpb.TraitName),
		)
	}

	return nil
}
