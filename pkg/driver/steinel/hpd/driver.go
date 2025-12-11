package hpd

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/steinel/hpd/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airqualitysensorpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

const DriverName = "steinel-hpd"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(services.Node),
		health:    services.Health,
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser[config.Root](config.ParseConfig),
	)
	d.logger = services.Logger.Named(DriverName)
	return d
}

type Driver struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	health    *healthpb.Checks
	logger    *zap.Logger

	client *Client

	airQualitySensor *AirQualitySensor
	occupancy        *Occupancy
	temperature      *TemperatureSensor

	udmiServiceServer *UdmiServiceServer
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	announcer := d.announcer.Replace(ctx)
	grp, ctx := errgroup.WithContext(ctx)

	d.client = NewInsecureClient(cfg.IpAddress, cfg.Password)

	d.airQualitySensor = NewAirQualitySensor(d.client, d.logger.Named("AirQualityValue").With(zap.String("ipAddress", cfg.IpAddress)))
	d.occupancy = NewOccupancySensor(d.client, d.logger.Named("Occupancy").With(zap.String("ipAddress", cfg.IpAddress)))
	d.temperature = NewTemperatureSensor(d.client, d.logger.Named("Temperature").With(zap.String("ipAddress", cfg.IpAddress)))
	d.udmiServiceServer = NewUdmiServiceServer(d.logger.Named("UdmiServiceServer"), d.airQualitySensor.AirQualityValue, d.occupancy.OccupancyValue, d.temperature.TemperatureValue, cfg.UDMITopicPrefix)

	announcer.Announce(cfg.Name,
		node.HasMetadata(cfg.Metadata),
		node.HasTrait(trait.AirQualitySensor, node.WithClients(airqualitysensorpb.WrapApi(d.airQualitySensor))),
		node.HasTrait(trait.OccupancySensor, node.WithClients(occupancysensorpb.WrapApi(d.occupancy))),
		node.HasTrait(trait.AirTemperature, node.WithClients(airtemperaturepb.WrapApi(d.temperature))),
		node.HasTrait(udmipb.TraitName, node.WithClients(gen.WrapUdmiService(d.udmiServiceServer))),
	)

	faultCheck, err := d.health.NewFaultCheck(cfg.Name, commsHealthCheck)
	if err != nil {
		d.logger.Error("failed to create health check", zap.String("device", cfg.Name), zap.Error(err))
		return err
	}

	poller := newPoller(d.client, cfg.PollInterval.Duration, d.logger.Named("SteinelPoller").With(zap.String("ipAddress", cfg.IpAddress)), faultCheck, d.airQualitySensor, d.occupancy, d.temperature)

	grp.Go(func() error {
		poller.startPoll(ctx)
		return nil
	})

	go func() {
		_ = grp.Wait() // won't error in current implementation
		faultCheck.Dispose()
		d.client.Client.CloseIdleConnections()
	}()

	return nil
}
