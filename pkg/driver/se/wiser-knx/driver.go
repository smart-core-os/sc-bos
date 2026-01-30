package wiser_knx

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/lightpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/modepb"
)

const DriverName = "se-wiser-knx"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		Services:  services,
		announcer: node.NewReplaceAnnouncer(services.Node),
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(ParseConfig),
	)
	d.logger = services.Logger.Named(DriverName)
	d.lightsByAddress = make(map[string]*lightpb.Model)
	d.modesByAddress = make(map[string]*modepb.Model)
	return d
}

type Driver struct {
	*service.Service[Config]
	driver.Services
	announcer *node.ReplaceAnnouncer
	logger    *zap.Logger

	cfg             Config
	client          *Client
	lightsByAddress map[string]*lightpb.Model
	modesByAddress  map[string]*modepb.Model
}

func (d *Driver) applyConfig(ctx context.Context, cfg Config) error {
	announcer := d.announcer.Replace(ctx)
	d.cfg = cfg

	// create a new client to communicate with the Wiser controller
	pass, err := cfg.Password.Read()
	if err != nil {
		return err
	}
	d.client = NewInsecureClient(cfg.Host, cfg.Username, pass)

	for _, dev := range cfg.Devices {
		if dev.Metadata != nil {
			announcer.Announce(dev.Name, node.HasMetadata(dev.Metadata))
		}

		if dev.Address == "" && len(dev.Addresses) == 0 {
			return fmt.Errorf("address or addresses is required")
		} else if dev.Address != "" && len(dev.Addresses) > 0 {
			return fmt.Errorf("address and addresses cannot both be specified")
		}

		if dev.Address != "" {
			dev.Addresses = map[string]string{"light": dev.Address}
		}

		_dev := dev
		for t, addr := range dev.Addresses {
			switch t {
			case "light":
				l := lightpb.NewModel()
				c := lightpb.WrapApi(lightServer{
					LightApiServer: lightpb.NewModelServer(l),
					client:         d.client,
					device:         &_dev,
					logger:         d.logger.With(zap.String("name", dev.Name)),
				})
				announcer.Announce(dev.Name, node.HasTrait(trait.Light, node.WithClients(c)))

				d.lightsByAddress[addr] = l
			case "override":
				modes := &traits.Modes{
					Modes: []*traits.Modes_Mode{
						&traits.Modes_Mode{
							Name:   "lighting.mode",
							Values: []*traits.Modes_Value{{Name: "auto"}, {Name: "manual"}},
						},
					},
				}

				modeModel := modepb.NewModelModes(modes)
				s := &modeInfoServer{
					Modes: &traits.ModesSupport{
						ModeValuesSupport: &types.ResourceSupport{
							Readable: true, Writable: true, Observable: true,
						},
						AvailableModes: modes,
					},
				}

				announcer.Announce(dev.Name, node.HasTrait(trait.Mode, node.WithClients(
					modepb.WrapApi(&modeServer{
						ModeApiServer: modepb.NewModelServer(modeModel),
						client:        d.client,
						device:        &_dev,
						logger:        d.logger.With(zap.String("name", dev.Name)),
					}),
					modepb.WrapInfo(s),
				)))

				d.modesByAddress[addr] = modeModel
			}
		}
	}

	go d.poll(ctx)

	return nil
}

func (d *Driver) poll(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.Poll.Duration)
	defer ticker.Stop()

	// update on initial load
	d.doPoll()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.doPoll()
		}
	}
}

func (d *Driver) doPoll() {
	// query all objects
	objects, err := QueryObjects(d.client)
	if err != nil {
		d.logger.Error("Error querying objects", zap.Error(err))
	}
	// loop through response objects
	for _, obj := range objects {
		// if matching device address
		if dev, ok := d.lightsByAddress[obj.Address]; ok {
			// update model brightness value for that device
			lvl := obj.Data.(float64)
			b := &traits.Brightness{
				LevelPercent: float32(lvl),
			}
			_, err = dev.UpdateBrightness(b)
			if err != nil {
				d.logger.Error("Error updating brightness", zap.Error(err))
			}
		} else if dev, ok := d.modesByAddress[obj.Address]; ok {
			var modeStr string
			b := obj.Data.(bool)
			if b {
				modeStr = "manual"
			} else {
				modeStr = "auto"
			}
			m := &traits.ModeValues{
				Values: map[string]string{"lighting.mode": modeStr},
			}
			_, err = dev.UpdateModeValues(m)
			if err != nil {
				d.logger.Error("Error updating mode", zap.Error(err))
			}
		}
	}
}

type lightServer struct {
	traits.LightApiServer
	client *Client
	device *Device
	logger *zap.Logger
}

func (l lightServer) UpdateBrightness(ctx context.Context, req *traits.UpdateBrightnessRequest) (*traits.Brightness, error) {
	err := SetValue(l.client, l.device.Addresses["light"], fmt.Sprintf("%f", req.Brightness.LevelPercent))
	if err != nil {
		return nil, err
	}

	brightness, err := l.LightApiServer.UpdateBrightness(ctx, req)
	if err != nil {
		return nil, err
	}
	return brightness, nil
}

type modeInfoServer struct {
	traits.UnimplementedModeInfoServer
	Modes *traits.ModesSupport
}

func (i *modeInfoServer) DescribeModes(context.Context, *traits.DescribeModesRequest) (*traits.ModesSupport, error) {
	return i.Modes, nil
}

type modeServer struct {
	traits.ModeApiServer
	client *Client
	device *Device
	logger *zap.Logger
}

func (m *modeServer) UpdateModeValues(ctx context.Context, req *traits.UpdateModeValuesRequest) (*traits.ModeValues, error) {
	val := req.ModeValues.Values["lighting.mode"] == "manual"

	err := SetValue(m.client, m.device.Addresses["override"], val)
	if err != nil {
		return nil, err
	}

	return m.ModeApiServer.UpdateModeValues(ctx, req)
}
