package config

import (
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	auto.Config
	// Schedule defines when to reset the enter/leave counter.
	// Defaults to midnight every day.
	Schedule *jsontypes.Schedule `json:"schedule,omitempty"`
	// Devices is the name of each device to reset.
	Devices []string `json:"devices,omitempty"`
}
