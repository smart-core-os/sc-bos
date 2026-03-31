package udmi

import (
	"context"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi/config"
)

// newMqttClient returns a new MQTT client from the given config
func newMqttClient(cfg config.Root) (mqtt.Client, error) {
	options, err := cfg.Broker.ClientOptions()
	if err != nil {
		return nil, err
	}
	return mqtt.NewClient(options), nil
}

// mqttPublisher is a Publisher backed by MQTT
func mqttPublisher(client mqtt.Client, qos byte, retained bool) Publisher {
	return PublisherFunc(func(ctx context.Context, topic string, payload any) error {
		token := client.Publish(topic, qos, retained, payload)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-token.Done():
			return token.Error()
		}
	})
}

// mqttPublisher is a Subscriber backed by MQTT
func mqttSubscriber(client mqtt.Client, qos byte) Subscriber {
	return SubscriberFunc(func(ctx context.Context, topic string, cb mqtt.MessageHandler) error {
		token := client.Subscribe(topic, qos, cb)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-token.Done():
			return token.Error()
		}
	})
}
