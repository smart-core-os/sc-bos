package config

import (
	"testing"
	"time"
)

func TestRead(t *testing.T) {
	valid := `{
		"type": "setpointhealth",
		"name": "fcu",
		"devices": [{"field": "metadata.traits.name", "stringEqual": "smartcore.traits.AirTemperature"}],
		"source": {
			"trait": "smartcore.traits.AirTemperature",
			"measured": "ambientTemperature.valueCelsius",
			"setPoint": "temperatureSetPoint.valueCelsius"
		},
		"tolerance": 1.5,
		"duration": "1h",
		"check": {"displayName": "Set point tracking"}
	}`

	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{name: "valid", config: valid, wantError: false},
		{name: "invalid json", config: `{not json`, wantError: true},
		{
			name: "zero tolerance",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"trait": "smartcore.traits.AirTemperature", "measured": "a", "setPoint": "b"},
				"duration": "1h"
			}`,
			wantError: true,
		},
		{
			name: "zero duration",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"trait": "smartcore.traits.AirTemperature", "measured": "a", "setPoint": "b"},
				"tolerance": 1.5
			}`,
			wantError: true,
		},
		{
			name: "missing measured",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"trait": "smartcore.traits.AirTemperature", "setPoint": "b"},
				"tolerance": 1.5, "duration": "1h"
			}`,
			wantError: true,
		},
		{
			name: "missing setPoint",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"trait": "smartcore.traits.AirTemperature", "measured": "a"},
				"tolerance": 1.5, "duration": "1h"
			}`,
			wantError: true,
		},
		{
			name: "missing trait",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"measured": "a", "setPoint": "b"},
				"tolerance": 1.5, "duration": "1h"
			}`,
			wantError: true,
		},
		{
			name: "unsupported trait",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"trait": "smartcore.traits.NotATrait", "measured": "a", "setPoint": "b"},
				"tolerance": 1.5, "duration": "1h"
			}`,
			wantError: true,
		},
		{
			name: "maxDuration greater than duration",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"trait": "smartcore.traits.AirTemperature", "measured": "a", "setPoint": "b"},
				"tolerance": 1.5, "duration": "1h", "maxDuration": "3h"
			}`,
			wantError: false,
		},
		{
			name: "maxDuration not greater than duration",
			config: `{
				"type": "setpointhealth", "name": "fcu",
				"source": {"trait": "smartcore.traits.AirTemperature", "measured": "a", "setPoint": "b"},
				"tolerance": 1.5, "duration": "1h", "maxDuration": "1h"
			}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read([]byte(tt.config))
			if tt.wantError && err == nil {
				t.Errorf("Read() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Read() unexpected error: %v", err)
			}
		})
	}
}

func TestReadParsesDurationAndTolerance(t *testing.T) {
	cfg, err := Read([]byte(`{
		"type": "setpointhealth", "name": "fcu",
		"source": {"trait": "smartcore.traits.AirTemperature", "measured": "a", "setPoint": "b"},
		"tolerance": 2.5, "duration": "30m"
	}`))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if cfg.Tolerance != 2.5 {
		t.Errorf("Tolerance = %v, want 2.5", cfg.Tolerance)
	}
	if cfg.Duration.Duration != 30*time.Minute {
		t.Errorf("Duration = %v, want 30m", cfg.Duration.Duration)
	}
}
