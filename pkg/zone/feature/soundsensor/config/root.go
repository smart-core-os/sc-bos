package config

import (
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	SoundSensors []string `json:"soundSensors,omitempty"`
}

