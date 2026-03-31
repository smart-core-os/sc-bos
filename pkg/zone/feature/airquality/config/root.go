package config

import (
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	AirQualitySensors []string `json:"airQualitySensors,omitempty"`
}
