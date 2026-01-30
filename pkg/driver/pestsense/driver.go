package pestsense

import (
	"context"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/driver/pestsense/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

const DriverName = "pestsense"

var Factory driver.Factory = factory{}

type factory struct{}

func (f factory) New(services driver.Services) service.Lifecycle {
	d := &Driver{
		announcer: node.NewReplaceAnnouncer(services.Node),
		logger:    services.Logger.Named(DriverName),
	}
	d.Service = service.New(
		service.MonoApply(d.applyConfig),
		service.WithParser(config.ParseConfig),
		service.WithOnStop[config.Root](d.onStop),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logCtx service.RetryContext) {
			logCtx.LogTo("applyConfig", d.logger)
		}), service.RetryWithMinDelay(10*time.Second)),
	)

	return d
}

type Driver struct {
	*service.Service[config.Root]
	logger    *zap.Logger
	announcer *node.ReplaceAnnouncer

	devices map[string]*pestSensor

	client       mqtt.Client
	currentTopic string
}

func (d *Driver) applyConfig(ctx context.Context, cfg config.Root) error {
	announcer := d.announcer.Replace(ctx)

	if d.client != nil && d.client.IsConnected() {
		if d.currentTopic != "" {
			token := d.client.Unsubscribe(d.currentTopic)
			token.Wait()
		}
		d.client.Disconnect(250)
	}

	// Connect to MQTT
	var err error
	d.client, err = newMqttClient(cfg)
	if err != nil {
		return err
	}

	connected := d.client.Connect()
	connected.Wait()
	if connected.Error() != nil {
		return connected.Error()
	}
	d.logger.Debug("connected")

	d.devices = make(map[string]*pestSensor)
	// Add devices and apis
	for _, device := range cfg.Devices {
		sensor := newPestSensor(device.Id, device.Name)
		d.devices[device.Id] = sensor

		announcer.Announce(device.Name,
			node.HasMetadata(device.Metadata),
			node.HasTrait(trait.OccupancySensor, node.WithClients(occupancysensorpb.WrapApi(sensor))))
	}

	var responseHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		handleResponse(msg.Payload(), d.devices, d.logger)
	}

	token := d.client.Subscribe(cfg.Broker.Topic, *cfg.Broker.QoS, responseHandler)
	token.Wait()
	if token.Error() != nil {
		d.logger.Error("failed to subscribe", zap.String("topic", cfg.Broker.Topic), zap.Error(token.Error()))
		return token.Error()
	}
	d.currentTopic = cfg.Broker.Topic
	d.logger.Debug("subscribed to topic", zap.String("topic", d.currentTopic))
	return nil
}

func newMqttClient(cfg config.Root) (mqtt.Client, error) {
	options, err := cfg.Broker.ClientOptions()
	if err != nil {
		return nil, err
	}
	return mqtt.NewClient(options), nil
}

func (d *Driver) onStop() {
	if d.client != nil {
		d.client.Disconnect(250)
	}
}
