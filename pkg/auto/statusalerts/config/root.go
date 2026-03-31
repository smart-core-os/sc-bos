package config

import (
	"encoding/json"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func ReadBytes(data []byte) (cfg Root, err error) {
	err = json.Unmarshal(data, &cfg)
	if cfg.Debounce != nil {
		for i, source := range cfg.Sources {
			if source.Debounce == nil {
				source.Debounce = cfg.Debounce
				cfg.Sources[i] = source
			}
		}
	}
	return
}

type Root struct {
	auto.Config
	// Name of the device that stores the alerts.
	// Must implement AlertAdminApi.
	Destination string `json:"destination,omitempty"`
	// If true, all devices on the current node that implement Status will be monitored.
	// Additional sources may be defined via Sources.
	DiscoverSources bool `json:"discoverSources,omitempty"`
	// Name of the devices that implement Status trait and that are monitored.
	Sources []Source `json:"sources,omitempty"`
	// Delay querying the status of devices by this much, to allow them to boot up.
	DelayStart *jsontypes.Duration `json:"delayStart,omitempty"`
	// Default debounce time for all sources.
	Debounce *jsontypes.Duration `json:"debounce,omitempty"`
	// Device name prefixes to ignore.
	// Only used if DiscoverSources is true.
	IgnorePrefixes []string `json:"ignorePrefixes,omitempty"`
}

type Source struct {
	Name      string `json:"name,omitempty"`
	Floor     string `json:"floor,omitempty"`
	Zone      string `json:"zone,omitempty"`
	Subsystem string `json:"subsystem,omitempty"`

	// Don't record alerts until after this time expires, reduces noise.
	Debounce *jsontypes.Duration `json:"debounce,omitempty"`
}

const DefaultDebounce = 15 * time.Second

func (s Source) DebounceOrDefault() time.Duration {
	if s.Debounce == nil {
		return DefaultDebounce
	}
	return s.Debounce.Duration
}
