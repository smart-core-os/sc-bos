package config

import (
	"github.com/smart-core-os/sc-bos/pkg/driver"
)

type Root struct {
	driver.BaseConfig
	LightingModes []string `json:"lightingModes"`
	HVACModes     []string `json:"hvacModes"`
}
