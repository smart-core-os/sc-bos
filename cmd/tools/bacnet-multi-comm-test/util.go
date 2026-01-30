package main

import (
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

func shouldDiscoverObjects(cfg config.Root, device config.Device) bool {
	if *discoverObjects {
		return true
	}
	if device.DiscoverObjects != nil {
		return *device.DiscoverObjects
	}
	return cfg.DiscoverObjects
}
