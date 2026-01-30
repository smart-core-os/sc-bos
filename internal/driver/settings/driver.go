package settings

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/internal/driver/settings/config"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/modepb"
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

	modes := &traits.Modes{}
	collectModes(modes, "lighting.mode", cfg.LightingModes...)
	collectModes(modes, "hvac.mode", cfg.HVACModes...)

	modeModel := modepb.NewModelModes(modes)
	info := &infoServer{
		Modes: &traits.ModesSupport{
			ModeValuesSupport: &types.ResourceSupport{
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

func collectModes(modes *traits.Modes, mode string, values ...string) {
	var modeValues []*traits.Modes_Value
	for _, value := range values {
		modeValues = append(modeValues, &traits.Modes_Value{
			Name: value,
		})
	}
	if len(modeValues) > 0 {
		modes.Modes = append(modes.Modes, &traits.Modes_Mode{
			Name:   mode,
			Values: modeValues,
		})
	}
}
