package udmi

import (
	"context"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// capturingClient embeds mqtt.Client so only Publish is implemented; it records
// the qos and retained flags passed to the most recent Publish call. Any other
// method call panics, which is fine as mqttPublisher only calls Publish.
type capturingClient struct {
	mqtt.Client
	qos      byte
	retained bool
}

func (c *capturingClient) Publish(_ string, qos byte, retained bool, _ any) mqtt.Token {
	c.qos = qos
	c.retained = retained
	return doneToken{}
}

// doneToken is an already-completed mqtt.Token with no error.
type doneToken struct{}

func (doneToken) Wait() bool                     { return true }
func (doneToken) WaitTimeout(time.Duration) bool { return true }
func (doneToken) Error() error                   { return nil }
func (doneToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func TestMqttPublisher_QoSByTopic(t *testing.T) {
	const eventQoS, stateQoS = 0, 1
	tests := map[string]byte{
		// telemetry (event topics) publish at eventQoS
		"client/site-01/HVAC/PICV-12345/event/pointset/points": eventQoS,
		"client/site-01/HVAC/PICV-12345/events/pointset":       eventQoS,
		// state and metadata publish at stateQoS
		"client/site-01/HVAC/PICV-12345/state":         stateQoS,
		"client/site-01/HVAC/PICV-12345/metadata.json": stateQoS,
		"client/site-01/HVAC/PICV-12345/metadata":      stateQoS,
	}
	for topic, wantQoS := range tests {
		client := &capturingClient{}
		pub := mqttPublisher(client, eventQoS, stateQoS, false)
		if err := pub.Publish(context.Background(), topic, "payload"); err != nil {
			t.Fatalf("Publish(%q) returned error: %v", topic, err)
		}
		if client.qos != wantQoS {
			t.Errorf("Publish(%q) used qos %d, want %d", topic, client.qos, wantQoS)
		}
	}
}

func TestIsEventTopic(t *testing.T) {
	tests := map[string]bool{
		// pointset/event topics: ephemeral, not retained
		"client/site-01/HVAC/PICV-12345/event/pointset/points": true,
		"client/site-01/HVAC/PICV-12345/events/pointset":       true,
		// state and metadata: retained
		"client/site-01/HVAC/PICV-12345/state":         false,
		"client/site-01/HVAC/PICV-12345/metadata.json": false,
		"client/site-01/HVAC/PICV-12345/metadata":      false,
	}
	for topic, want := range tests {
		if got := isEventTopic(topic); got != want {
			t.Errorf("isEventTopic(%q) = %v, want %v", topic, got, want)
		}
	}
}
