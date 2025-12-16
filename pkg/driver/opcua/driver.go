package opcua

import (
	"context"
	"fmt"
	"time"

	"github.com/gopcua/opcua"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/transport"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/electricpb"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
)

const DriverName = "opcua"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	logger := services.Logger.Named(DriverName)

	d := &Driver{
		announcer: node.NewReplaceAnnouncer(services.Node),
		health:    services.Health,
		logger:    logger,
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(config.ReadBytes),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", logger)
		}), service.RetryWithMinDelay(5*time.Second), service.RetryWithInitialDelay(5*time.Second)),
	)
	return d
}

type Driver struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	health    *healthpb.Checks
	logger    *zap.Logger
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {

	a := d.announcer.Replace(ctx)
	healthSystemName = cfg.SystemHealth.SystemName

	systemCheck, err := d.health.NewFaultCheck(cfg.Name, getSystemHealthCheck(cfg.SystemHealth.OccupantImpact, cfg.SystemHealth.EquipmentImpact))
	if err != nil {
		d.logger.Warn("failed to create system fault check", zap.Error(err))
		return err
	}

	opcClient, err := d.connectOpcClient(ctx, cfg, systemCheck)
	if err != nil {
		return err
	}

	systemCheck.ClearFaults()
	client := NewClient(opcClient, d.logger, cfg.Conn.SubscriptionInterval.Duration, cfg.Conn.ClientId)

	if cfg.Meta != nil {
		a.Announce(cfg.Name, node.HasMetadata(cfg.Meta))
	}

	grp, ctx := errgroup.WithContext(ctx)
	var errs error
	for _, dev := range cfg.Devices {

		faultCheck, err := d.health.NewFaultCheck(dev.Name, getDeviceHealthCheck(cfg.SystemHealth.OccupantImpact, cfg.SystemHealth.EquipmentImpact))
		if err != nil {
			d.logger.Error("failed to create device fault check", zap.String("device", dev.Name), zap.Error(err))
			return err
		}

		var allFeatures []node.Feature
		opcDev := newDevice(&dev, d.logger, client, faultCheck)

		for _, t := range dev.Traits {
			switch t.Kind {
			case meter.TraitName:
				opcDev.meter, err = newMeter(dev.Name, t, d.logger)
				if err != nil {
					errs = fmt.Errorf("failed to add trait for device %s: %w", dev.Name, err)
				} else {
					allFeatures = append(allFeatures, node.HasTrait(meter.TraitName, node.WithClients(gen.WrapMeterApi(opcDev.meter), gen.WrapMeterInfo(opcDev.meter))))
				}
			case transport.TraitName:
				opcDev.transport, err = newTransport(dev.Name, t, d.logger)
				if err != nil {
					errs = fmt.Errorf("failed to add trait for device %s: %w", dev.Name, err)
				} else {
					allFeatures = append(allFeatures, node.HasTrait(transport.TraitName, node.WithClients(gen.WrapTransportApi(opcDev.transport), gen.WrapTransportInfo(opcDev.transport))))
				}
			case udmipb.TraitName:
				opcDev.udmi, err = newUdmi(dev.Name, t, d.logger)
				if err != nil {
					errs = fmt.Errorf("failed to add trait for device %s: %w", dev.Name, err)
				} else {
					allFeatures = append(allFeatures, node.HasTrait(udmipb.TraitName, node.WithClients(gen.WrapUdmiService(opcDev.udmi))))
				}
			case trait.Electric:
				opcDev.electric, err = newElectric(dev.Name, t, d.logger)
				if err != nil {
					errs = fmt.Errorf("failed to add trait for device %s: %w", dev.Name, err)
				} else {
					allFeatures = append(allFeatures, node.HasTrait(trait.Electric, node.WithClients(electricpb.WrapApi(opcDev.electric))))
				}
			default:
				d.logger.Error("unknown trait", zap.String("trait", t.Name))
			}
		}

		if dev.Meta != nil {
			allFeatures = append(allFeatures, node.HasMetadata(dev.Meta))
		}

		if errs != nil {
			d.logger.Error("errors encountered whilst loading driver", zap.String("device", dev.Name), zap.Error(errs))
		}

		a.Announce(dev.Name, allFeatures...)
		grp.Go(func() error {
			return opcDev.subscribe(ctx)
		})
	}

	go func() {
		err := grp.Wait()
		d.logger.Error("run error", zap.Error(err))
		_ = opcClient.Close(ctx)
	}()
	return nil
}

func (d *Driver) connectOpcClient(ctx context.Context, cfg config.Root, faultCheck *healthpb.FaultCheck) (*opcua.Client, error) {
	rel := &gen.HealthCheck_Reliability{}
	opcClient, err := opcua.NewClient(cfg.Conn.Endpoint)
	if err != nil {
		rel.State = gen.HealthCheck_Reliability_UNRELIABLE
		rel.LastError = &gen.HealthCheck_Error{
			SummaryText: "Internal Driver Error",
			DetailsText: "The device has an unrecognised internal status code",
			Code:        statusToHealthCode(DriverConfigError),
		}
		faultCheck.UpdateReliability(ctx, rel)
		d.logger.Error("error creating new client", zap.Error(err))
		return nil, err
	}

	err = opcClient.Connect(ctx)
	if err != nil {
		rel.State = gen.HealthCheck_Reliability_NO_RESPONSE
		rel.LastError = &gen.HealthCheck_Error{
			SummaryText: "Server Unreachable",
			DetailsText: "The opcua server is unreachable",
			Code:        statusToHealthCode(ServerUnreachable),
		}
		faultCheck.UpdateReliability(ctx, rel)
		d.logger.Error("error connecting to opc ua server", zap.Error(err))
		return nil, err
	}

	return opcClient, nil
}
