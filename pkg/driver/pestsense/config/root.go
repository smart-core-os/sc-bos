package config

import (
	"encoding/json"
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
)

// Default values for config fields
const (
	// DefaultQoS is the default MQTT Quality of Service level (1 = at least once delivery)
	DefaultQoS = 1
)

type Root struct {
	driver.BaseConfig

	Broker  MQTTBroker `json:"broker,omitempty"`
	Devices []Device   `json:"devices,omitempty"`
}

type Device struct {
	Name string `json:"name"`
	Id   string `json:"id,omitempty"`

	Metadata *traits.Metadata `json:"metadata,omitempty"`
}

type MQTTBroker struct {
	Host     string `json:"host,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Topic    string `json:"topic,omitempty"`
	// QoS is the MQTT Quality of Service level (0, 1, or 2), defaults to 1
	QoS *byte `json:"qos,omitempty"`
}

// ParseConfig parses the JSON configuration data into a Root struct and sets default values for optional fields.
//
// It returns the parsed Root configuration and any error encountered during parsing.
func ParseConfig(data []byte) (Root, error) {
	root := Root{}

	err := json.Unmarshal(data, &root)
	if err != nil {
		return Root{}, err
	}

	if root.Broker.QoS == nil {
		root.Broker.QoS = new(byte)
		*root.Broker.QoS = DefaultQoS
	} else if *root.Broker.QoS > 2 || *root.Broker.QoS < 0 {
		return Root{}, fmt.Errorf("invalid MQTT QoS level in config: %d", *root.Broker.QoS)
	}

	return root, nil
}

func (b MQTTBroker) ClientOptions() (*mqtt.ClientOptions, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.Host)
	opts.SetUsername(b.Username)
	opts.SetOrderMatters(false)
	opts.SetPassword(b.Password)
	return opts, nil
}
