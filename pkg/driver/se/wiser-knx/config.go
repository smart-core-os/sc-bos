package wiser_knx

import (
	"encoding/json"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func DefaultConfig() Config {
	return Config{
		Poll:     jsontypes.Duration{Duration: 10 * time.Second},
		Username: "remote",
		Password: jsontypes.Password{
			PasswordFile: "/run/secrets/wiser-knx-password",
		},
	}
}

func ParseConfig(raw []byte) (Config, error) {
	parsed := DefaultConfig()
	err := json.Unmarshal(raw, &parsed)
	return parsed, err
}

// Config is the configuration for the Wiser for KNX driver.
type Config struct {
	driver.BaseConfig

	// The IP address of the Wiser for KNX controller.
	Host string `json:"host"`

	// The username to use when connecting to the Wiser for KNX controller (default: "remote").
	Username string `json:"username,omitempty"`
	jsontypes.Password

	// The poll interval to use when polling the Wiser for KNX controller (default: 10 seconds).
	Poll jsontypes.Duration `json:"poll,omitempty"`

	// The list of exported objects on the Wiser for KNX controller.
	Devices []Device `json:"devices,omitempty"`
}

// Device config for an object on the Wiser for KNX controller.
// Note: only 1 of Address or Addresses should be specified.
type Device struct {
	// The device name (e.g. "inf/sc-01/lights/flex-room-01")
	Name string `json:"name"`
	// The address of the object on the Wiser for KNX controller (e.g. "1/1/1")
	Address string `json:"address,omitempty"`
	// Map of device component to address (e.g. {"light": "1/1/1"}). Possible components:
	// - "light": the light object
	// - "override": an optional bool object that disables the Wiser's automation when true - exposed using the Mode trait.
	Addresses map[string]string `json:"addresses,omitempty"`
	// The metadata associated with the device.
	Metadata *traits.Metadata `json:"metadata,omitempty"`
}
