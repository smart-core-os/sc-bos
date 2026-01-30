package azureiot

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

const (
	DefaultPollInterval = 5 * time.Minute
	DefaultBacklogSize  = 1000 // in messages per device
)

type Config struct {
	// Azure IoT Hub dps credentials, used for devices where DeviceConfig.ConnectionString == ""
	GroupKey     string `json:"groupKey"`
	GroupKeyFile string `json:"groupKeyFile"`
	IDScope      string `json:"idScope"`

	Devices []DeviceConfig `json:"devices"`

	// When a device doesn't support pullTraits, use this poll interval.
	// Defaults to DefaultPollInterval, can't be less than minPollInterval.
	PollInterval *jsontypes.Duration `json:"pollInterval"`
	// When present, provides defaults for connecting to all smart core devices.
	RemoteNode *RemoteNodeConfig `json:"remoteNode,omitempty"`
}

// DeviceConfig represents an Azure IoT Hub device.
// This device may map to one or more Smart Core devices.
type DeviceConfig struct {
	SCDeviceConfig                  // config used when an IoT Hub device maps to one Smart Core device
	Children       []SCDeviceConfig `json:"children,omitempty"` // config used when an IoT Hub device maps to multiple Smart Core devices
	// todo: DiscoverDevices bool // if true, discover devices from Smart Core and use them as the Children list

	// Azure IoT Hub connection details, one of these must be provided
	RegistrationID       string `json:"registrationID,omitempty"`       // Device Provisioning Service registration ID - usually becomes the device name in the cloud
	ConnectionString     string `json:"connectionString,omitempty"`     // Use a connection string to directly connect to IoT Hub, bypassing Device Provisioning Service
	ConnectionStringFile string `json:"connectionStringFile,omitempty"` // Filesystem path to a file containing the connection string
}

// UsesConnectionString returns if the device will connect directly using a Connection String, bypassing DPS.
//
// This is true if either ConnectionString or ConnectionStringFile is non-empty.
func (dc *DeviceConfig) UsesConnectionString() bool {
	return dc.ConnectionString != "" || dc.ConnectionStringFile != ""
}

// SCDeviceConfig represents a Smart Core device information will be pulled from.
type SCDeviceConfig struct {
	Name                string       `json:"name"`                // Smart Core name to monitor
	Traits              []trait.Name `json:"traits"`              // Traits to pullTraits from
	IgnoreUnknownTraits bool         `json:"ignoreUnknownTraits"` // Ignore unknown traits instead of erroring.
	// todo: empty traits list means the intersection of device and auto supported traits

	PollInterval *jsontypes.Duration `json:"pollInterval"`         // Defaults to Config.PollInterval
	RemoteNode   *RemoteNodeConfig   `json:"remoteNode,omitempty"` // Defaults to Config.RemoteNode, if absent (or Host=="") uses local device resolution.
}

type RemoteNodeConfig struct {
	Host                 string `json:"host,omitempty"` // "host[:port]" of the Smart Core node, port defaults to 23557
	*jsontypes.TLSConfig        // How to connect to the remote host, defaults to hub TLS config if enrolled.
}

const minPollInterval = 5 * time.Second // minimum rate of sending data to IoT Hub

func ParseConfig(jsonBytes []byte) (Config, error) {
	cfg := DefaultConfig()
	err := json.Unmarshal(jsonBytes, &cfg)

	if i := cfg.PollInterval.Or(DefaultPollInterval); i < minPollInterval {
		return cfg, fmt.Errorf("pollInterval %v must be at least %v", i, minPollInterval)
	}
	if rn := cfg.RemoteNode; rn != nil {
		if rn.Host != "" {
			rn.Host = hostWithDefaultPort(rn.Host, 23557)
		}
	}

	setSCDefaults := func(deviceCfg *SCDeviceConfig) {
		if deviceCfg.Name == "" {
			return
		}
		if deviceCfg.PollInterval == nil {
			deviceCfg.PollInterval = cfg.PollInterval
		}
		switch {
		case deviceCfg.RemoteNode == nil:
			deviceCfg.RemoteNode = cfg.RemoteNode
		case deviceCfg.RemoteNode.Host != "":
			deviceCfg.RemoteNode.Host = hostWithDefaultPort(deviceCfg.RemoteNode.Host, 23557)
		default:
			deviceCfg.RemoteNode = nil // override using Config.RemoteNode
		}
	}

	for i, deviceCfg := range cfg.Devices {
		if !deviceCfg.UsesConnectionString() {
			// 	needsGroupKey = true

			// if the group key is used, then an ID scope is also mandatory
			if cfg.IDScope == "" {
				return cfg, fmt.Errorf("id scope is required when using group keys")
			}
		}
		if deviceCfg.ConnectionString != "" && deviceCfg.ConnectionStringFile != "" {
			return cfg, fmt.Errorf("device %q has connectionString and connectionStringFile - only one is permitted", deviceCfg.Name)
		}

		setSCDefaults(&deviceCfg.SCDeviceConfig)
		for i, child := range deviceCfg.Children {
			setSCDefaults(&child)
			deviceCfg.Children[i] = child
		}

		cfg.Devices[i] = deviceCfg
	}

	return cfg, err
}

func DefaultConfig() Config {
	return Config{
		PollInterval: &jsontypes.Duration{Duration: DefaultPollInterval},
	}
}

func hostWithDefaultPort(s string, p int) string {
	if s == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(s); err != nil {
		return net.JoinHostPort(s, fmt.Sprintf("%d", p))
	}
	return s
}

var Blocks = []block.Block{
	{
		Path: []string{"devices"},
		Key:  "name",
		Blocks: []block.Block{
			{Path: []string{"children"}, Key: "name"},
			{Path: []string{"registrationID"}},
			{Path: []string{"connectionString"}},
		},
	},
	{Path: []string{"remoteNode"}},
}
