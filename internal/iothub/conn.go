package iothub

import (
	"context"
	"fmt"
	"io"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/smart-core-os/sc-bos/internal/iothub/auth"
	"github.com/smart-core-os/sc-bos/internal/util/mqttutil"
)

// Conn represents a connection to IoT Hub.
type Conn interface {
	// SendOutputMessage sends telemetry as json encode payload to the device event topic.
	SendOutputMessage(ctx context.Context, telemetry any) error
	io.Closer
}

type conn struct {
	mqtt           mqtt.Client
	telemetryTopic string
}

// Dial returns a Conn connected to the host described by params.
func Dial(ctx context.Context, params ConnectionParameters) (Conn, error) {
	mqttOpts, err := MQTTClientOptions(params.HostName, params.DeviceID, "", &auth.LocalSigner{Secret: params.SharedAccessKey})
	if err != nil {
		return nil, err
	}
	mqttClient, err := mqttutil.Connect(ctx, mqttOpts)
	if err != nil {
		return nil, err
	}

	return &conn{
		mqtt:           mqttClient,
		telemetryTopic: fmt.Sprintf("devices/%s/messages/events/$.ct=application%%2Fjson&$.ce=utf-8", params.DeviceID),
	}, nil
}

func (c *conn) SendOutputMessage(ctx context.Context, telemetry any) error {
	return mqttutil.SendJSON(ctx, c.mqtt, c.telemetryTopic, telemetry)
}

func (c *conn) Close() error {
	c.mqtt.Disconnect(0)
	return nil
}
