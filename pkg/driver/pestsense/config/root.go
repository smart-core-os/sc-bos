package config

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
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
}

func (b MQTTBroker) ClientOptions() (*mqtt.ClientOptions, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.Host)
	opts.SetUsername(b.Username)
	opts.SetOrderMatters(false)
	opts.SetPassword(b.Password)
	return opts, nil
}
