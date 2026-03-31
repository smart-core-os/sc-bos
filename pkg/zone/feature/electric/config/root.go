package config

import (
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	Electrics      []string            `json:"electrics,omitempty"`
	ElectricGroups map[string][]string `json:"electricGroups,omitempty"`
}
