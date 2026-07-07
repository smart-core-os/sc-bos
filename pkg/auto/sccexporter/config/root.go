package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// DefaultTopicPrefix is the fixed intent prefix under which telemetry is
// published (the publish-authz scope). Everything below it is source-native
// UDMI addressing. See docs/connect-telemetry-ingest.md.
const DefaultTopicPrefix = "tlm"

type Mqtt struct {
	Host        string `json:"host"`                  // MQTT v5 broker address, e.g. "tls://broker.example.com:8883"
	TopicPrefix string `json:"topicPrefix,omitempty"` // fixed intent prefix, defaults to "tlm"
	ClientId    string `json:"clientId,omitempty"`    // MQTT client id

	// Credential mode. Exactly one of useCloudCredential or the file-path certs
	// below is used. useCloudCredential draws the Connect leaf certificate from
	// the node's cloud connection (mTLS, identity via enrichment). The file-path
	// certs are a dev/test fallback usable before the cloud credential is wired.
	UseCloudCredential bool   `json:"useCloudCredential,omitempty"`
	ClientKeyPath      string `json:"clientKeyPath,omitempty"`  // file path to client private key
	ClientCertPath     string `json:"clientCertPath,omitempty"` // file path to client certificate
	CaCertPath         string `json:"caCertPath,omitempty"`     // optional CA for verifying the broker; empty ⇒ system roots

	Qos              *int                `json:"qos,omitempty"`                       // MQTT qos, defaults to 1 if not provided or not 0,1 or 2
	ConnectTimeout   *jsontypes.Duration `json:"connectTimeout,omitempty,omitzero"`   // timeout for connecting to MQTT broker, defaults to 5s
	PublishTimeout   *jsontypes.Duration `json:"publishTimeout,omitempty,omitzero"`   // timeout for publishing to MQTT, defaults to 5s
	SendInterval     *jsontypes.Schedule `json:"sendInterval,omitempty,omitzero"`     // time between sends, defaults to 15m
	MetadataInterval *int                `json:"metadataInterval,omitempty,omitzero"` // how often to publish discovery (every N data sends), defaults to 100
}

type Root struct {
	auto.Config

	// Traits is the set of traits to export. Devices implementing one of these
	// traits are discovered and polled. Only smartcore.bos.Meter is supported today.
	Traits []string `json:"traits"`
	Mqtt   Mqtt     `json:"mqtt"`
	// PointNaming selects the point key emitted on the wire: "dbo" (default) emits DBO
	// standard field names (so a building-config translation is an identity mapping);
	// "raw" emits the raw Smart Core point names (usage/produced) and leaves the
	// name→field mapping to the consumer (Connect's per-source pipeline).
	PointNaming string `json:"pointNaming,omitempty"`
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

	if root.Mqtt.Host == "" {
		return Root{}, fmt.Errorf("config parse failed, mqtt.host is required")
	}

	if root.Mqtt.TopicPrefix == "" {
		root.Mqtt.TopicPrefix = DefaultTopicPrefix
	}

	switch root.PointNaming {
	case "":
		root.PointNaming = "dbo"
	case "dbo", "raw":
		// ok
	default:
		return Root{}, fmt.Errorf("config parse failed, pointNaming must be \"dbo\" or \"raw\", got %q", root.PointNaming)
	}

	// Credential modes are mutually exclusive: either the cloud credential, or
	// the file-path certs, never both.
	if root.Mqtt.UseCloudCredential {
		if root.Mqtt.ClientCertPath != "" || root.Mqtt.ClientKeyPath != "" || root.Mqtt.CaCertPath != "" {
			return Root{}, fmt.Errorf("config parse failed, mqtt.useCloudCredential cannot be combined with clientCertPath/clientKeyPath/caCertPath")
		}
	} else {
		if root.Mqtt.ClientCertPath == "" || root.Mqtt.ClientKeyPath == "" {
			return Root{}, fmt.Errorf("config parse failed, either mqtt.useCloudCredential must be set or both mqtt.clientCertPath and mqtt.clientKeyPath must be provided")
		}
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
