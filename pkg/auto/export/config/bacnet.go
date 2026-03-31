package config

import (
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

type BacnetSource struct {
	Source
	COV     *config.COV    `json:"cov,omitempty"`
	Devices []BacnetDevice `json:"devices,omitempty"`

	// PrintTiming when true causes the publishing for this source to print timing statistics to the log each time
	PrintTiming bool `json:"printTiming,omitempty"`
}

type BacnetDevice struct {
	Name    string         `json:"name,omitempty"`
	Objects []BacnetObject `json:"objects,omitempty"`
}

type BacnetObject struct {
	ID         config.ObjectID     `json:"id"`
	Properties []config.PropertyID `json:"properties,omitempty"`
}
