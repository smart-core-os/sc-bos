package resourceutilisation

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	// PollInterval controls how often resource stats are collected.
	// Defaults to 10s if unset.
	PollInterval *jsontypes.Duration `json:"pollInterval,omitempty"`
}

func (r Root) pollInterval() time.Duration {
	return r.PollInterval.Or(10 * time.Second)
}
