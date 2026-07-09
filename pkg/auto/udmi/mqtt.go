package udmi

import (
	"context"
	"strings"

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

// mqttPublisher is a Publisher backed by MQTT.
//
// Retention is decided per topic: UDMI pointset/event topics carry ephemeral
// telemetry and are published unretained so a (re)connecting subscriber doesn't
// replay a stale value as if it were live; state and metadata describe
// last-known device status and model and are retained so subscribers see them
// immediately on connect. retainAll (the auto's "retained" config) forces the
// retained flag on for every topic, preserving the legacy retain-everything
// behaviour.
//
// QoS is decided by the same split: event topics publish at eventQoS (telemetry,
// typically at-most-once) while state/metadata topics publish at stateQoS.
func mqttPublisher(client mqtt.Client, eventQoS, stateQoS byte, retainAll bool) Publisher {
	return PublisherFunc(func(ctx context.Context, topic string, payload any) error {
		event := isEventTopic(topic)
		qos := eventQoS
		if !event {
			qos = stateQoS
		}
		retained := retainAll || !event
		token := client.Publish(topic, qos, retained, payload)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-token.Done():
			return token.Error()
		}
	})
}

// isEventTopic reports whether an MQTT topic is a UDMI event topic. Event topics
// use an "/event" or "/events" segment (e.g. ".../event/pointset/points",
// ".../events/pointset"); state ("/state") and metadata ("/metadata.json")
// topics do not.
func isEventTopic(topic string) bool {
	return strings.Contains(topic, "/event")
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
