package config

import (
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/block/mdblock"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

type Root struct {
	driver.BaseConfig
	Devices []Device `json:"devices,omitempty"`
}

type Device struct {
	*metadatapb.Metadata
}

var Blocks = []block.Block{
	{
		Path:   []string{"devices"},
		Key:    "name",
		Blocks: mdblock.Categories,
	},
}
