package config

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// DefaultHeartbeatInterval is how long a source may stay quiet before the auto
// asks it for a current message to publish.
const DefaultHeartbeatInterval = 4 * time.Hour

// DefaultMinSendInterval is the shortest time allowed between two publishes on the
// same pointset event topic. Zero leaves the floor off, so every change a source
// reports is published, which is the behaviour predating the setting.
const DefaultMinSendInterval time.Duration = 0

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
	// HeartbeatInterval is the longest a source may go without publishing a pointset
	// event. Once it has been quiet for this long it is asked, via GetExportMessage,
	// for a current message to publish, so consumers can tell a stable device from a
	// dead one. Sources only emit on change, so a device whose readings never move is
	// otherwise silent indefinitely.
	//
	// The message published is one the source collected and stamped itself; the auto
	// never replays or restamps. A source that cannot produce one answers Unavailable
	// and nothing is published, so silence still means dead. A source that doesn't
	// implement GetExportMessage at all answers Unimplemented, which disarms its
	// heartbeat for the lifetime of the auto.
	//
	// Defaults to 4h; set "0s" to disable.
	HeartbeatInterval *jsontypes.Duration `json:"heartbeatInterval,omitempty,omitzero"`
	// MinSendInterval is the shortest time allowed between two publishes on the same
	// pointset event topic: a rate limit for chatty devices. Sources emit on every
	// change they observe, so a device polled every 10s whose readings never settle
	// publishes every 10s, indefinitely.
	//
	// A change arriving inside the interval isn't dropped, it's held: the newest held
	// payload for the topic is published as soon as the interval expires, so consumers
	// see the current value at a bounded rate. Intermediate values are lost, which is
	// what a rate limit means — don't set this on a topic whose every sample matters.
	//
	// This is a floor on publishing, not a change-of-value deadband: it bounds how
	// often a value can be reported, not how much it must move to be worth reporting.
	// Defaults to 0, which is off.
	MinSendInterval *jsontypes.Duration `json:"minSendInterval,omitempty,omitzero"`
}
