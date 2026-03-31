package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Mqtt struct {
	Agent            string              `json:"agent"`
	Host             string              `json:"host"`
	Topic            string              `json:"topic"` // the topic to publish to
	ClientId         string              `json:"clientId"`
	ClientKeyPath    string              `json:"clientKeyPath"`                       // file path to client private key
	ClientCertPath   string              `json:"clientCertPath"`                      // file path to client certificate
	CaCertPath       string              `json:"caCertPath"`                          // file path to CA certificate
	ConnectTimeout   *jsontypes.Duration `json:"connectTimeout,omitempty,omitzero"`   // timeout for connecting to MQTT broker, defaults to 5s
	PublishTimeout   *jsontypes.Duration `json:"publishTimeout,omitempty,omitzero"`   // timeout for publishing to MQTT, defaults to 5s
	Qos              *int                `json:"qos,omitempty"`                       // MQTT qos, defaults to 1 if not provided or not 0,1 or 2
	SendInterval     *jsontypes.Schedule `json:"sendInterval,omitempty,omitzero"`     // time between sends, defaults to 15m
	MetadataInterval *int                `json:"metadataInterval,omitempty,omitzero"` // how often to include metadata (every N data sends), defaults to 100
}

type Root struct {
	auto.Config

	// A list of all traits we want to send data for
	Traits []string `json:"traits"`
	Mqtt   Mqtt     `json:"mqtt"`
	// FetchTimeout is the maximum time to wait for a single device's trait data fetch
	// If a device takes longer than this, the fetch is cancelled and the device is skipped
	// Default is 5 seconds
	FetchTimeout *jsontypes.Duration `json:"fetchTimeout,omitempty,omitzero"`
}

func ParseConfig(data []byte) (Root, error) {
	root := Root{}

	if err := json.Unmarshal(data, &root); err != nil {
		return Root{}, err
	}

	if root.Mqtt.SendInterval == nil {
		root.Mqtt.SendInterval = jsontypes.MustParseSchedule("*/15 * * * *")
	}
	if root.Mqtt.ConnectTimeout == nil || root.Mqtt.ConnectTimeout.Duration == 0 {
		root.Mqtt.ConnectTimeout = &jsontypes.Duration{Duration: 5 * time.Second}
	}
	if root.Mqtt.PublishTimeout == nil || root.Mqtt.PublishTimeout.Duration == 0 {
		root.Mqtt.PublishTimeout = &jsontypes.Duration{Duration: 5 * time.Second}
	}
	if root.Mqtt.Qos == nil {
		q := 1
		root.Mqtt.Qos = &q
	} else if *root.Mqtt.Qos < 0 || *root.Mqtt.Qos > 2 {
		return Root{}, fmt.Errorf("config parse failed, mqtt.qos must be 0, 1, or 2")
	}
	if root.Mqtt.MetadataInterval == nil {
		interval := 100
		root.Mqtt.MetadataInterval = &interval
	}
	if root.FetchTimeout == nil || root.FetchTimeout.Duration == 0 {
		root.FetchTimeout = &jsontypes.Duration{Duration: 5 * time.Second}
	}

	return root, nil
}
