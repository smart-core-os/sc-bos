package config

import (
	"github.com/smart-core-os/sc-bos/pkg/auto"
)

type Root struct {
	auto.Config

	// Broker configures an MQTT broker to export data to, and subscribe to topics on.
	Broker *MQTTBroker `json:"broker,omitempty"`

	// When true the auto will inspect the local node for all devices that can export UDMI information.
	// Additional sources can be configured using "sources".
	DiscoverSources bool `json:"discoverSources,omitempty"`
	// the names to use for rpc requests to UdmiService
	Sources []string `json:"sources,omitempty"`
}
