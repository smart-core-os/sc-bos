package config

import (
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/block/mdblock"
	"github.com/smart-core-os/sc-bos/pkg/driver"
)

type Root struct {
	driver.BaseConfig
	Devices []Device `json:"devices,omitempty"`
}

type Device struct {
	*traits.Metadata
}

var Blocks = []block.Block{
	{
		Path:   []string{"devices"},
		Key:    "name",
		Blocks: mdblock.Categories,
	},
}
