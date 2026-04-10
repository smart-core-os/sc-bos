package hpd

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/steinel/hpd/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
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
	brightnessSensor *brightnessSensor
	occupancy        *Occupancy
	soundSensor      *soundSensor
	temperature      *TemperatureSensor

	udmiServiceServer *UdmiServiceServer
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	announcer := d.announcer.Replace(ctx)
	grp, ctx := errgroup.WithContext(ctx)

	d.client = newInsecureClient(cfg.IpAddress, cfg.Password)

	d.airQualitySensor = newAirQualitySensor(d.client, d.logger.Named("AirQualityValue").With(zap.String("ipAddress", cfg.IpAddress)))
	d.brightnessSensor = newBrightnessSensor(d.client, d.logger.Named("Brightness").With(zap.String("ipAddress", cfg.IpAddress)))
	d.occupancy = newOccupancySensor(d.client, d.logger.Named("Occupancy").With(zap.String("ipAddress", cfg.IpAddress)))
	d.soundSensor = newSoundSensor(d.client, d.logger.Named("soundSensor").With(zap.String("ipAddress", cfg.IpAddress)))
	d.temperature = newTemperatureSensor(d.client, d.logger.Named("Temperature").With(zap.String("ipAddress", cfg.IpAddress)))
	d.udmiServiceServer = newUdmiServiceServer(d.logger.Named("UdmiServiceServer"), d.airQualitySensor.AirQualityValue, d.occupancy.OccupancyValue, d.temperature.TemperatureValue, cfg.UDMITopicPrefix)

	announcer.Announce(cfg.Name,
		node.HasMetadata(cfg.Metadata),
		node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(d.airQualitySensor)),
		node.HasTrait(trait.AirQualitySensor),
		node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(d.temperature)),
		node.HasTrait(trait.AirTemperature),
		node.HasServer(brightnesssensorpb.RegisterBrightnessSensorApiServer, brightnesssensorpb.BrightnessSensorApiServer(d.brightnessSensor)),
		node.HasTrait(trait.BrightnessSensor),
		node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(d.occupancy)),
		node.HasTrait(trait.OccupancySensor),
		node.HasServer(soundsensorpb.RegisterSoundSensorApiServer, soundsensorpb.SoundSensorApiServer(d.soundSensor)),
		node.HasTrait(soundsensorpb.TraitName),
		node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(d.udmiServiceServer)),
		node.HasTrait(udmipb.TraitName),
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
