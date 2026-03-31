package driver

import (
	"encoding/json"

	"github.com/smart-core-os/sc-bos/internal/util/jsonutil"
)

type BaseConfig struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Disabled bool   `json:"disabled,omitempty"`
}

type RawConfig struct {
	BaseConfig
	Raw json.RawMessage `json:"-"`
}

func (c *RawConfig) MarshalJSON() ([]byte, error) {
	// override "name", "type" and "disabled" from BaseConfig
	return jsonutil.MarshalObjects(c.Raw, c.BaseConfig)
}

func (c *RawConfig) UnmarshalJSON(buf []byte) error {
	c.Raw = buf
	return json.Unmarshal(buf, &c.BaseConfig)
}
