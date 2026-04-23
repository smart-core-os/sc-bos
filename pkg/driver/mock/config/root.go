package config

import (
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/block/mdblock"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

type Root struct {
	driver.BaseConfig
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
	Devices     []Device     `json:"devices,omitempty"`
}

// HealthCheck configures a simulated driver-level connectivity health check for UI testing.
type HealthCheck struct {
	// FaultProbability is the probability (0.0–1.0) that each tick results in a fault state.
	// The remaining probability results in a healthy state. Defaults to 0.15.
	FaultProbability float64 `json:"faultProbability,omitempty"`
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
