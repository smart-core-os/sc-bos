package settings

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/driver/settings/config"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/modepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const DriverName = "settings"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		services:  services,
		announcer: node.NewReplaceAnnouncer(services.Node),
		logger:    services.Logger.Named("settings"),
	}
	d.Service = service.New(service.MonoApply(d.applyConfig))
	return d
}

type Driver struct {
	*service.Service[config.Root]
	services  driver.Services
	announcer *node.ReplaceAnnouncer

	logger *zap.Logger
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	announcer := d.announcer.Replace(ctx)

	modes := &modepb.Modes{}
	collectModes(modes, "lighting.mode", cfg.LightingModes...)
	collectModes(modes, "hvac.mode", cfg.HVACModes...)

	modeModel := modepb.NewModelModes(modes)
	info := &infoServer{
		Modes: &modepb.ModesSupport{
			ModeValuesSupport: &typespb.ResourceSupport{
				Readable: true, Writable: true, Observable: true,
			},
			AvailableModes: modes,
		},
	}

	announcer.Announce(cfg.Name, node.HasTrait(trait.Mode, node.WithClients(
		modepb.WrapApi(modepb.NewModelServer(modeModel)),
		modepb.WrapInfo(info),
	)))

	return nil
}

func collectModes(modes *modepb.Modes, mode string, values ...string) {
	var modeValues []*modepb.Modes_Value
	for _, value := range values {
		modeValues = append(modeValues, &modepb.Modes_Value{
			Name: value,
		})
	}
	if len(modeValues) > 0 {
		modes.Modes = append(modes.Modes, &modepb.Modes_Mode{
			Name:   mode,
			Values: modeValues,
		})
	}
}
