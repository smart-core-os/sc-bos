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
	// Retained, when true, publishes every message with the MQTT retained flag set.
	// When false (the default), only state and metadata are retained; pointset/event
	// topics are always published unretained so subscribers get real-time telemetry
	// rather than a replayed stale value.
	Retained bool `json:"retained,omitempty"`
	// QoS is the MQTT Quality of Service level (0, 1, or 2) used for publishing
	// telemetry (pointset event topics) and for all subscriptions. Defaults to 0
	// (at-most-once) when unset.
	QoS byte `json:"qos,omitempty"`
	// StateQoS is the MQTT QoS level used for publishing state and metadata topics
	// (everything that is not an event topic). Defaults to 0 (matching QoS) when
	// unset, preserving the previous single-QoS behaviour.
	StateQoS byte `json:"stateQos,omitempty"`
}
