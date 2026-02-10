package config

import (
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	ReadOnlyLights    bool                `json:"readOnlyLights,omitempty"`
	Lights            []string            `json:"lights,omitempty"`            // Announces as {zone}
	LightGroups       map[string][]string `json:"lightGroups,omitempty"`       // Announced as {zone}/{key}
	ConcurrentUpdates *int                `json:"concurrentUpdates,omitempty"` // Number of concurrent updates to send to the platform. Default is 1 (no concurrency).
}
