package config

import (
	"github.com/smart-core-os/sc-bos/pkg/system"
)

type Root struct {
	system.Config

	// Ignore contains a list of enrolled host:port that we should not proxy.
	// Useful if you set up more than one gateway enrolled with the same hub.
	// This controller will always ignore its own endpoint.
	Ignore []string `json:"ignore,omitempty"`

	// HubMode dictates how the gateway should connect to the hub. This will be "remote" for systems where the gateway is
	// not running on the same host as the hub (default behaviour), and "local" where the gateway is also the hub.
	HubMode string `json:"hubMode,omitempty"`
}

const (
	HubModeRemote = "remote" // default
	HubModeLocal  = "local"
)
