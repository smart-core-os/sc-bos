package hpd

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/steinel/hpd/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
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

	// devicesStopped waits until the devices set up by the previous applyConfig have
	// stopped and released their health checks. nil before the first apply.
	devicesStopped func()
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	if d.devicesStopped != nil {
		// MonoApply has cancelled the previous apply's context, but its devices release
		// their health checks asynchronously as they stop. Wait for them so re-registering
		// the checks below doesn't fail with ErrAlreadyExists.
		d.devicesStopped()
	}
	announcer := d.announcer.Replace(ctx)

	var wg sync.WaitGroup
	d.devicesStopped = wg.Wait

	var errs []error
	for _, devCfg := range cfg.Devices {
		if err := d.applyDeviceConfig(ctx, announcer, &wg, devCfg); err != nil {
			// one bad device shouldn't stop the others from running
			d.logger.Error("failed to set up device", zap.String("device", devCfg.Name), zap.Error(err))
			errs = append(errs, fmt.Errorf("%s: %w", devCfg.Name, err))
		}
	}
	if len(errs) > 0 && len(errs) == len(cfg.Devices) {
		// no device made it, surface that via the service status
		return errors.Join(errs...)
	}
	return nil
}

func (d *Driver) applyDeviceConfig(ctx context.Context, announcer node.Announcer, wg *sync.WaitGroup, cfg config.Device) error {
	logger := d.logger.With(zap.String("device", cfg.Name), zap.String("ipAddress", cfg.IpAddress))
	client := newInsecureClient(cfg.IpAddress, cfg.ResolvedPassword())

	airQualitySensor := newAirQualitySensor(client, logger.Named("AirQualityValue"))
	brightnessSensor := newBrightnessSensor(client, logger.Named("Brightness"))
	occupancy := newOccupancySensor(client, logger.Named("Occupancy"))
	soundSensor := newSoundSensor(client, logger.Named("soundSensor"))
	temperature := newTemperatureSensor(client, logger.Named("Temperature"))

	faultCheck, err := d.health.NewFaultCheck(cfg.Name, commsHealthCheck())
	if err != nil {
		return fmt.Errorf("failed to create health check: %w", err)
	}

	features := []node.Feature{
		node.HasMetadata(cfg.Metadata),
		node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airQualitySensor)),
		node.HasTrait(trait.AirQualitySensor),
		node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(temperature)),
		node.HasTrait(trait.AirTemperature),
		node.HasServer(brightnesssensorpb.RegisterBrightnessSensorApiServer, brightnesssensorpb.BrightnessSensorApiServer(brightnessSensor)),
		node.HasTrait(trait.BrightnessSensor),
		node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancy)),
		node.HasTrait(trait.OccupancySensor),
		node.HasServer(soundsensorpb.RegisterSoundSensorApiServer, soundsensorpb.SoundSensorApiServer(soundSensor)),
		node.HasTrait(soundsensorpb.TraitName),
		node.HasDeviceType(metadatapb.Metadata_DEVICE),
	}
	if cfg.UDMITopicPrefix != "" {
		// without a topic prefix devices would all export to the same MQTT topic, so no prefix means no UDMI
		udmiServiceServer := newUdmiServiceServer(logger.Named("UdmiServiceServer"),
			airQualitySensor.AirQualityValue, occupancy.OccupancyValue, temperature.TemperatureValue, cfg.UDMITopicPrefix)
		features = append(features,
			node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(udmiServiceServer)),
			node.HasTrait(udmipb.TraitName),
		)
	}
	announcer.Announce(cfg.Name, features...)

	poller := newPoller(client, cfg.PollInterval.Or(config.DefaultPollInterval), logger.Named("SteinelPoller"), faultCheck, airQualitySensor, occupancy, temperature)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer client.Client.CloseIdleConnections()
		defer faultCheck.Dispose()
		poller.startPoll(ctx)
	}()

	return nil
}
