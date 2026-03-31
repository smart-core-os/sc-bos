package config

import (
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	OpenClose       []string            `json:"openClose,omitempty"`
	OpenCloseGroups map[string][]string `json:"openCloseGroups,omitempty"`
}
