package config

import (
	"encoding/json"

	"go.uber.org/multierr"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config
	Self
	Raw json.RawMessage `json:"-"`
}

type Self struct {
	Metadata *traits.Metadata   `json:"metadata,omitempty"`
	Drivers  []driver.RawConfig `json:"drivers,omitempty"`
}

func (r *Root) UnmarshalJSON(buf []byte) error {
	r.Raw = buf
	return multierr.Combine(
		json.Unmarshal(buf, &r.Config),
		json.Unmarshal(buf, &r.Self),
	)
}
