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
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/gen"
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
		logger:    logger,
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(config.ReadBytes),
		service.WithRetry[config.Root](
			service.RetryWithLogger(func(logContext service.RetryContext) {
				logContext.LogTo("applyConfig", logger)
			}),
			// Use config timing values for retry policy
			// These defaults match config.ReadBytes defaults (BackoffStart: 2s, BackoffMax: 30s)
			service.RetryWithInitialDelay(2*time.Second),
			service.RetryWithMinDelay(2*time.Second),
			service.RetryWithMaxDelay(30*time.Second),
		),
	)
	return d
}

type Driver struct {
	*service.Service[config.Root]
	logger    *zap.Logger
	announcer *node.ReplaceAnnouncer
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {

	a := d.announcer.Replace(ctx)

	opcClient, err := opcua.NewClient(cfg.Conn.Endpoint)
	if err != nil {
		d.logger.Warn("NewClient error", zap.Error(err))
		return err
	}

	err = opcClient.Connect(ctx)
	if err != nil {
		d.logger.Warn("Connect error", zap.Error(err))
		return err
	}

	client := NewClient(opcClient, d.logger, cfg.Conn.SubscriptionInterval.Duration, cfg.Conn.ClientId)

	if cfg.Meta != nil {
		a.Announce(cfg.Name, node.HasMetadata(cfg.Meta))
	}

	grp, ctx := errgroup.WithContext(ctx)
	for _, dev := range cfg.Devices {
		allFeatures := []node.Feature{node.HasMetadata(dev.Meta)}
		opcDev := newDevice(&dev, d.logger, client)

		for _, t := range dev.Traits {
			switch t.Kind {
			case meter.TraitName:
				opcDev.meter, err = newMeter(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.String("trait", string(meter.TraitName)), zap.Error(err))
					return err
				}
				allFeatures = append(allFeatures, node.HasTrait(meter.TraitName, node.WithClients(gen.WrapMeterApi(opcDev.meter), gen.WrapMeterInfo(opcDev.meter))))

			case transport.TraitName:
				opcDev.transport, err = newTransport(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.String("trait", string(transport.TraitName)), zap.Error(err))
					return err
				}
				allFeatures = append(allFeatures, node.HasTrait(transport.TraitName, node.WithClients(gen.WrapTransportApi(opcDev.transport), gen.WrapTransportInfo(opcDev.transport))))

			case udmipb.TraitName:
				opcDev.udmi, err = newUdmi(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.String("trait", string(udmipb.TraitName)), zap.Error(err))
					return err
				}
				allFeatures = append(allFeatures, node.HasTrait(udmipb.TraitName, node.WithClients(gen.WrapUdmiService(opcDev.udmi))))

			case trait.Electric:
				opcDev.electric, err = newElectric(dev.Name, t, d.logger)
				if err != nil {
					d.logger.Error("failed to add trait, invalid config", zap.String("device", dev.Name), zap.String("trait", string(trait.Electric)), zap.Error(err))
					return err
				}
				allFeatures = append(allFeatures, node.HasTrait(trait.Electric, node.WithClients(electricpb.WrapApi(opcDev.electric))))

			default:
				d.logger.Error("unknown trait", zap.String("trait", t.Name))
			}
		}

		a.Announce(dev.Name, allFeatures...)
		dev := opcDev
		grp.Go(func() error {
			return dev.run(ctx)
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
