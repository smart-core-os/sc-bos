package config

import (
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	AccessPoints []string `json:"accessPoints,omitempty"`
}
