package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// DefaultPollInterval is how often each device is polled for updates unless configured otherwise.
const DefaultPollInterval = 60 * time.Second

type Root struct {
	driver.BaseConfig

	// PasswordFile is a shared default for all devices, can be overridden per device.
	PasswordFile string `json:"passwordFile,omitempty"`
	// PollInterval is the interval between polling each device for updates, defaults to 60 seconds.
	// Can be overridden per device.
	PollInterval *jsontypes.Duration `json:"pollInterval,omitempty,omitzero"`

	// Devices lists the HPD sensors managed by this driver.
	Devices []Device `json:"devices,omitempty"`

	// Legacy single-device config. When ipAddress is set the root describes one device
	// announced using the driver name. Mutually exclusive with Devices.

	// Metadata is the smart core metadata associated with the legacy single device.
	// Deprecated: use Devices and set metadata per device instead.
	Metadata *metadatapb.Metadata `json:"metadata,omitempty"`
	// IpAddress is the address of the legacy single device.
	// Deprecated: use Devices and set ipAddress per device instead.
	IpAddress string `json:"ipAddress,omitempty"`
	// Password is rejected when set, plaintext passwords in config are not supported.
	// Deprecated: use PasswordFile.
	Password string `json:"password,omitempty"`
	// UDMITopicPrefix is the UDMI/MQTT topic prefix of the legacy single device.
	// Deprecated: use Devices and set udmiTopicPrefix per device instead.
	UDMITopicPrefix string `json:"udmiTopicPrefix,omitempty"`
}

type Device struct {
	// Name is the smart core name the device is announced as.
	Name string `json:"name"`
	// Metadata is the smart core metadata associated with this device.
	Metadata *metadatapb.Metadata `json:"metadata,omitempty"`

	IpAddress string `json:"ipAddress"`
	// Password is rejected when set, plaintext passwords in config are not supported.
	// Deprecated: use PasswordFile.
	Password string `json:"password,omitempty"`
	// PasswordFile names a file the device password is read from, defaults to the root PasswordFile.
	PasswordFile string `json:"passwordFile,omitempty"`
	// PollInterval overrides the root pollInterval for this device.
	PollInterval *jsontypes.Duration `json:"pollInterval,omitempty,omitzero"`

	// UDMITopicPrefix supports the UDMI/MQTT automation.
	// When empty the device is not announced with the UDMI trait.
	// Must be unique between devices, otherwise their exports would share an MQTT topic.
	UDMITopicPrefix string `json:"udmiTopicPrefix,omitempty"`

	// password is the device password read from PasswordFile, see ResolvedPassword.
	password string
}

// ResolvedPassword returns the device password read from PasswordFile during ParseConfig.
func (d Device) ResolvedPassword() string {
	return d.password
}

func ParseConfig(data []byte) (Root, error) {
	root := Root{}

	err := json.Unmarshal(data, &root)
	if err != nil {
		return Root{}, err
	}

	if root.Password != "" {
		return Root{}, fmt.Errorf("plaintext passwords in config are not supported, use passwordFile")
	}
	if root.IpAddress != "" {
		if len(root.Devices) > 0 {
			return Root{}, fmt.Errorf("ipAddress and devices are mutually exclusive")
		}
		// legacy single-device config: the root describes one device named after the driver
		root.Devices = []Device{{
			Name:            root.Name,
			Metadata:        root.Metadata,
			IpAddress:       root.IpAddress,
			UDMITopicPrefix: root.UDMITopicPrefix,
		}}
	} else {
		if len(root.Devices) == 0 {
			return Root{}, fmt.Errorf("one of ipAddress or devices is required")
		}
		if root.Metadata != nil || root.UDMITopicPrefix != "" {
			// these don't cascade to devices, reject them rather than silently ignoring them
			return Root{}, fmt.Errorf("metadata and udmiTopicPrefix are legacy single-device options, set them per device")
		}
	}

	if root.PollInterval == nil || root.PollInterval.Duration == 0 {
		root.PollInterval = &jsontypes.Duration{Duration: DefaultPollInterval}
	}

	seenNames := make(map[string]struct{}, len(root.Devices))
	seenTopicPrefixes := make(map[string]struct{}, len(root.Devices))
	passwords := make(map[string]string) // file path -> contents, so files shared between devices are read once
	for i := range root.Devices {
		device := &root.Devices[i]
		if device.Name == "" {
			return Root{}, fmt.Errorf("devices[%d]: name is required", i)
		}
		if _, ok := seenNames[device.Name]; ok {
			return Root{}, fmt.Errorf("devices[%d]: duplicate name %q", i, device.Name)
		}
		seenNames[device.Name] = struct{}{}
		if device.IpAddress == "" {
			return Root{}, fmt.Errorf("devices[%d] %q: ipAddress is required", i, device.Name)
		}
		if device.Password != "" {
			return Root{}, fmt.Errorf("devices[%d] %q: plaintext passwords in config are not supported, use passwordFile", i, device.Name)
		}
		if err := device.resolvePassword(root.PasswordFile, passwords); err != nil {
			return Root{}, fmt.Errorf("devices[%d] %q: %w", i, device.Name, err)
		}
		if device.PollInterval == nil || device.PollInterval.Duration == 0 {
			device.PollInterval = root.PollInterval
		}
		if device.UDMITopicPrefix != "" {
			if _, ok := seenTopicPrefixes[device.UDMITopicPrefix]; ok {
				return Root{}, fmt.Errorf("devices[%d] %q: duplicate udmiTopicPrefix %q, devices would share an MQTT topic", i, device.Name, device.UDMITopicPrefix)
			}
			seenTopicPrefixes[device.UDMITopicPrefix] = struct{}{}
		}
	}

	return root, nil
}

// resolvePassword fills in d.password from d.PasswordFile, falling back to defaultFile.
// passwords caches file contents by path so files shared between devices are read once.
func (d *Device) resolvePassword(defaultFile string, passwords map[string]string) error {
	file := d.PasswordFile
	if file == "" {
		file = defaultFile
	}
	if file == "" {
		return fmt.Errorf("passwordFile is required")
	}
	if pass, ok := passwords[file]; ok {
		d.password = pass
		return nil
	}
	bs, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read password file: %w", err)
	}
	pass := strings.TrimSpace(string(bs))
	if pass == "" {
		return fmt.Errorf("password file %q is empty", file)
	}
	passwords[file] = pass
	d.password = pass
	return nil
}
