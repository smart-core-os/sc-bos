// Package opcua implements a Smart Core driver for OPC UA servers.
// It subscribes to OPC UA variable nodes and exposes their values through Smart Core traits
// including Meter, Electric, Transport, and UDMI.
//
// The driver creates an internal device instance for each configured device, which manages
// OPC UA subscriptions and routes value changes to the appropriate trait handlers.
package opcua

import (
	"context"
	"errors"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const DriverName = "opcua"

var Factory driver.Factory = factory{}

type factory struct{}

type EventHandler interface {
	handleEvent(ctx context.Context, node *ua.NodeID, value any)
}

func (f factory) New(services driver.Services) service.Lifecycle {
	logger := services.Logger.Named(DriverName)

	d := &Driver{
		announcer:   node.NewReplaceAnnouncer(services.Node),
		health:      services.Health,
		logger:      logger,
		systemCheck: services.SystemCheck,
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(config.ParseConfig),
		service.WithOnStop[config.Root](d.onStop),
		service.WithRetry[config.Root](
			service.RetryWithLogger(func(logContext service.RetryContext) {
				logContext.LogTo("applyConfig", logger)
			}),
			service.RetryWithInitialDelay(2*time.Second),
			service.RetryWithMinDelay(2*time.Second),
			service.RetryWithMaxDelay(30*time.Second),
		),
	)
	return d
}

type Driver struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	health    *healthpb.Checks
	logger    *zap.Logger

	systemCheck service.SystemCheck
	checks      []*healthpb.FaultCheck
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	a := d.announcer.Replace(ctx)

	d.dispose()

	opcClient, err := d.connectOpcClient(ctx, cfg)
	if err != nil {
		d.logger.Warn("Connect error", zap.Error(err))
		return err
	}

	client := NewClient(opcClient, d.logger, cfg.Conn.SubscriptionInterval.Duration, cfg.Conn.ClientId)

	a.Announce(cfg.Name, node.HasMetadata(cfg.Meta))

	grp, ctx := errgroup.WithContext(ctx)
	for _, dev := range cfg.Devices {
		allFeatures := []node.Feature{node.HasMetadata(dev.Meta)}

		faultCheck, err := d.health.NewFaultCheck(dev.Name, getDeviceHealthCheck(dev.Health.OccupantImpact.ToProto(), dev.Health.EquipmentImpact.ToProto()))
		if err != nil {
			d.logger.Error("failed to create device fault check", zap.String("device", dev.Name), zap.Error(err))
			return err
		}
		d.checks = append(d.checks, faultCheck)

		opcDev := newDevice(&dev, d.logger, client, faultCheck, d.systemCheck)

		for _, t := range dev.Traits {
			switch t.Kind {
			case meterpb.TraitName:
				m, err := newMeter(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", meterpb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, m)
				allFeatures = append(allFeatures,
					node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(m)),
					node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(m)),
					node.HasTrait(meterpb.TraitName),
				)

			case transportpb.TraitName:
				tr, err := newTransport(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", transportpb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, tr)
				allFeatures = append(allFeatures,
					node.HasServer(transportpb.RegisterTransportApiServer, transportpb.TransportApiServer(tr)),
					node.HasServer(transportpb.RegisterTransportInfoServer, transportpb.TransportInfoServer(tr)),
					node.HasTrait(transportpb.TraitName),
				)

			case udmipb.TraitName:
				u, err := newUdmi(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", udmipb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, u)
				allFeatures = append(allFeatures,
					node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(u)),
					node.HasTrait(udmipb.TraitName),
				)

			case trait.Electric:
				e, err := newElectric(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", trait.Electric), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, e)
				allFeatures = append(allFeatures,
					node.HasServer(electricpb.RegisterElectricApiServer, electricpb.ElectricApiServer(e)),
					node.HasTrait(trait.Electric),
				)

			case healthpb.TraitName:
				h, err := newHealth(t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.Stringer("trait", healthpb.TraitName), zap.Error(err))
					return err
				}
				opcDev.eventHandlers = append(opcDev.eventHandlers, h)
				for _, check := range h.cfg.Checks {
					c := getDeviceErrorCheck(check)
					fc, err := d.health.NewFaultCheck(dev.Name, c)
					if err != nil {
						d.logger.Error("failed to create health fault check", zap.String("device", dev.Name), zap.String("check", check.Id), zap.Error(err))
						return err
					}
					h.errorChecks[check.Id] = fc
					d.checks = append(d.checks, fc)
				}
			default:
				d.logger.Error("unknown trait", zap.String("trait", t.Name))
			}
		}

		a.Announce(dev.Name, allFeatures...)
		dev := opcDev
		grp.Go(func() error {
			return dev.subscribe(ctx)
		})
	}

	go func() {
		err := grp.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			d.logger.Error("run error", zap.Error(err))
		}

		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = opcClient.Close(closeCtx); err != nil {
			d.logger.Warn("failed to close opcua client", zap.Error(err))
		}
	}()
	return nil
}

func (d *Driver) connectOpcClient(ctx context.Context, cfg config.Root) (*opcua.Client, error) {
	opcClient, err := opcua.NewClient(cfg.Conn.Endpoint)
	if err != nil {
		if d.systemCheck != nil {
			d.systemCheck.MarkFailed(err)
		}
		d.logger.Error("error creating new client", zap.Error(err))
		return nil, err
	}

	err = opcClient.Connect(ctx)
	if err != nil {
		if d.systemCheck != nil {
			d.systemCheck.MarkFailed(err)
		}
		d.logger.Error("error connecting to opc ua server", zap.Error(err))
		return nil, err
	}
	if d.systemCheck != nil {
		d.systemCheck.MarkRunning()
	}
	return opcClient, nil
}

func (d *Driver) onStop() {
	d.dispose()
}

func (d *Driver) dispose() {
	if d.systemCheck != nil {
		d.systemCheck.Dispose()
	}

	for _, c := range d.checks {
		if c != nil {
			c.Dispose()
		}
	}
}
