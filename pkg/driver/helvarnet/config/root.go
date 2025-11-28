package config

import (
	"encoding/json"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// Default values for config fields
const (
	DefaultConnectTimeout    = 1 * time.Second
	DefaultRefreshOccupancy  = 10 * time.Second
	DefaultRefreshStatus     = 1 * time.Minute
	DefaultRxBufferSize      = 1024
	DefaultRxBufferSizeMin   = 1024
	DefaultRxBufferSizeMax   = 65536
	DefaultRxTimeout         = 1 * time.Second
	DefaultTxTimeout         = 5 * time.Millisecond
	DefaultSendPacketTimeout = 5 * time.Millisecond
	DefaultPort              = ":50000"
)

// Root represents the root configuration for the HelvarNet driver.
type Root struct {
	driver.BaseConfig

	ConnectTimeout *jsontypes.Duration `json:"connectTimeout,omitempty"`

	EmergencyLights []*Device `json:"emergencyLights,omitempty"`
	Lights          []*Device `json:"lights,omitempty"`
	LightingGroups  []*Device `json:"lightingGroups,omitempty"`
	Pirs            []*Device `json:"pirs,omitempty"`
	// RefreshOccupancy is the duration at which the pir sensors refresh their occupancy status
	// Defaults to every 10 seconds
	RefreshOccupancy *jsontypes.Duration `json:"refreshOccupancy,omitempty,omitzero"`
	// RefreshStatus is the duration between each command to query the device state
	// Defaults to every 1 min
	RefreshStatus *jsontypes.Duration `json:"refreshStatus,omitempty,omitzero"`
	// RxBufferSize is the size of the receive buffer for the TCP connection, defaults to 1024 bytes
	RxBufferSize *int `json:"rxBufferSize,omitempty,omitzero"`
	// RxTimeout is the duration to wait for a response from querying the device, defaults to 1 second
	RxTimeout *jsontypes.Duration `json:"rxTimeout,omitempty,omitzero"`
	// TxTimeout is the send timeout, defaults to 5 milliseconds
	TxTimeout *jsontypes.Duration `json:"txTimeout,omitempty,omitzero"`
	// SendPacketTimeout is the timeout used for sending packets, defaults to 5 milliseconds
	SendPacketTimeout *jsontypes.Duration `json:"sendPacketTimeout,omitempty,omitzero"`
	// Port is the TCP port used to connect to Helvarnet devices, defaults to ":50000"
	Port *string `json:"port,omitempty"`
	// RetrySleepDuration is the duration to wait before retrying a failed operation, defaults 500 microseconds
	RetrySleepDuration *jsontypes.Duration `json:"retrySleepDuration,omitempty,omitzero"`
}

// Device represents a HelvarNet device, which can be a light, lighting group, or PIR sensor.
//
// Name is the Smart Core device name.
// Address is the Helvarnet device address, in the format <cluster>.<router>.<subnet>.<device>.(<subdevice> - optional).
// GroupNumber is the Helvarnet group number.
// IpAddress is the device's IP address.
// Meta contains additional metadata for the device.
// DurationTestLength is the length of the duration test for emergency lights, if known.
// TopicPrefix is the topic prefix to use for the UDMI automation, without the trailing '/'. If empty, the device name will be used.
type Device struct {
	Name               string              `json:"name,omitempty"`
	Address            string              `json:"address,omitempty"`
	GroupNumber        *int                `json:"groupNumber,omitempty"`
	IpAddress          string              `json:"ipAddress,omitempty"`
	Meta               *traits.Metadata    `json:"meta,omitempty"`
	DurationTestLength *jsontypes.Duration `json:"durationTestLength,omitempty,omitzero"`
	TopicPrefix        string              `json:"topicPrefix,omitempty"`
}

// Scene represents a HelvarNet lighting scene, which is a combination of a block (address), scene, and title.
//
// Block is the address block for the scene.
// Scene is the scene identifier.
// Title is a human-readable title for the scene.
type Scene struct {
	Block string `json:"block,omitempty"`
	Scene string `json:"scene,omitempty"`
	Title string `json:"title,omitempty"`
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

	if root.ConnectTimeout == nil {
		root.ConnectTimeout = &jsontypes.Duration{Duration: DefaultConnectTimeout}
	}

	if root.RefreshOccupancy == nil {
		root.RefreshOccupancy = &jsontypes.Duration{Duration: DefaultRefreshOccupancy}
	}

	if root.RefreshStatus == nil {
		root.RefreshStatus = &jsontypes.Duration{Duration: DefaultRefreshStatus}
	}

	if root.RxTimeout == nil {
		root.RxTimeout = &jsontypes.Duration{Duration: DefaultRxTimeout}
	}

	if root.RxBufferSize == nil || *root.RxBufferSize < DefaultRxBufferSizeMin || *root.RxBufferSize > DefaultRxBufferSizeMax {
		root.RxBufferSize = new(int)
		*root.RxBufferSize = DefaultRxBufferSize
	}

	if root.TxTimeout == nil {
		root.TxTimeout = &jsontypes.Duration{Duration: DefaultTxTimeout}
	}

	if root.SendPacketTimeout == nil {
		root.SendPacketTimeout = &jsontypes.Duration{Duration: DefaultSendPacketTimeout}
	}

	if root.Port == nil {
		root.Port = new(string)
		*root.Port = DefaultPort
	}

	if root.RetrySleepDuration == nil {
		root.RetrySleepDuration = &jsontypes.Duration{Duration: 500 * time.Microsecond}
	}

	for _, device := range root.EmergencyLights {
		if device.TopicPrefix == "" {
			device.TopicPrefix = device.Name
		}
	}
	for _, device := range root.Lights {
		if device.TopicPrefix == "" {
			device.TopicPrefix = device.Name
		}
	}
	for _, device := range root.LightingGroups {
		if device.TopicPrefix == "" {
			device.TopicPrefix = device.Name
		}
	}
	for _, device := range root.Pirs {
		if device.TopicPrefix == "" {
			device.TopicPrefix = device.Name
		}
	}

	return root, nil
}
