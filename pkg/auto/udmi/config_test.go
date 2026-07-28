package udmi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi/config"
)

// The service parses config with plain json.Unmarshal (no SetDefaults hook), so
// the default has to come from Or at the point of use.
func TestConfig_HeartbeatInterval(t *testing.T) {
	tests := map[string]struct {
		raw  string
		want time.Duration
	}{
		"absent defaults to 4h": {raw: `{}`, want: config.DefaultHeartbeatInterval},
		"explicit value":        {raw: `{"heartbeatInterval":"30m"}`, want: 30 * time.Minute},
		"zero disables":         {raw: `{"heartbeatInterval":"0s"}`, want: 0},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var cfg config.Root
			if err := json.Unmarshal([]byte(tt.raw), &cfg); err != nil {
				t.Fatalf("unmarshal %s: %v", tt.raw, err)
			}
			if got := cfg.HeartbeatInterval.Or(config.DefaultHeartbeatInterval); got != tt.want {
				t.Errorf("heartbeat interval is %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := map[string]struct {
		raw     string
		wantErr bool
	}{
		"empty is valid":               {raw: `{}`},
		"heartbeat interval":           {raw: `{"heartbeatInterval":"1h"}`},
		"heartbeat disabled":           {raw: `{"heartbeatInterval":"0s"}`},
		"negative heartbeat interval":  {raw: `{"heartbeatInterval":"-1h"}`, wantErr: true},
		"qos out of range":             {raw: `{"qos":3}`, wantErr: true},
		"state qos out of range":       {raw: `{"stateQos":3}`, wantErr: true},
		"qos and heartbeat both valid": {raw: `{"qos":1,"heartbeatInterval":"4h"}`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var cfg config.Root
			if err := json.Unmarshal([]byte(tt.raw), &cfg); err != nil {
				t.Fatalf("unmarshal %s: %v", tt.raw, err)
			}
			err := validateConfig(cfg)
			if tt.wantErr && err == nil {
				t.Errorf("validateConfig(%s) returned nil, want an error", tt.raw)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateConfig(%s) returned %v, want nil", tt.raw, err)
			}
		})
	}
}
