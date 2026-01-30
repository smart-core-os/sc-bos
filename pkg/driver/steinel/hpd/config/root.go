package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	driver.BaseConfig

	// Smart core metadata associated with this device.
	Metadata *traits.Metadata `json:"metadata,omitempty"`

	IpAddress    string `json:"ipAddress"`
	Password     string `json:"password,omitempty"`
	PasswordFile string `json:"passwordFile,omitempty"`
	// PollInterval is the interval between polling the device for updates, defaults to 60 seconds
	PollInterval *jsontypes.Duration `json:"pollInterval,omitempty,omitzero"`

	// to support UDMI/MQTT automation
	UDMITopicPrefix string `json:"udmiTopicPrefix,omitempty"`
}

func ParseConfig(data []byte) (Root, error) {
	root := Root{}

	err := json.Unmarshal(data, &root)
	if err != nil {
		return Root{}, err
	}

	if root.IpAddress == "" {
		return Root{}, fmt.Errorf("ipAddress is required")
	}

	if root.Password == "" {
		bs, err := os.ReadFile(root.PasswordFile)
		if err != nil {
			return Root{}, fmt.Errorf("failed to read password file: %w", err)
		}
		root.Password = strings.TrimSpace(string(bs))
	}

	if root.PollInterval == nil || root.PollInterval.Duration == 0 {
		root.PollInterval = &jsontypes.Duration{Duration: time.Second * 60}
	}

	return root, nil
}
