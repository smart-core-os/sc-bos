package config

import (
	"strings"
	"testing"
)

func TestValidate_Trait(t *testing.T) {
	base := `{
		"type": "healthbounds",
		"name": "fcu",
		"source": {%s}
	}`

	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{name: "supported trait", source: `"trait": "smartcore.traits.AirTemperature"`, wantError: false},
		{name: "missing trait", source: `"resource": "AirTemperature"`, wantError: true},
		{name: "empty trait", source: `"trait": ""`, wantError: true},
		{name: "unsupported trait", source: `"trait": "smartcore.traits.NotATrait"`, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := []byte(strings.Replace(base, "%s", tt.source, 1))
			_, err := Read(cfg)
			if tt.wantError && err == nil {
				t.Errorf("Read() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Read() unexpected error: %v", err)
			}
		})
	}
}
