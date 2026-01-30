package auto

import (
	"encoding/json"

	"github.com/smart-core-os/sc-bos/internal/util/jsonutil"
)

type Config struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Disabled bool   `json:"disabled,omitempty"`
}

type RawConfig struct {
	Config
	Raw json.RawMessage `json:"-"`
}

func (c *RawConfig) MarshalJSON() ([]byte, error) {
	// override "name", "type" and "disabled" from Config
	return jsonutil.MarshalObjects(c.Raw, c.Config)
}

func (c *RawConfig) UnmarshalJSON(buf []byte) error {
	c.Raw = buf
	return json.Unmarshal(buf, &c.Config)
}
