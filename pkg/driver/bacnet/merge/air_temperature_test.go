package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

func float32Ptr(v float32) *float32 { return &v }

func TestUpdateMode(t *testing.T) {
	// Helper to create modeDataPoints
	newModeDataPoints := func(fanOn, heatingOn, coolingOn *float32) *modeDataPoints {
		return &modeDataPoints{
			FanOnValue:     fanOn,
			HeatingOnValue: heatingOn,
			CoolingOnValue: coolingOn,
		}
	}
	// Helper to create airTempModeConfig with thresholds
	newModeConfig := func(fanThresh, heatThresh, coolThresh *float32) *airTempModeConfig {
		return &airTempModeConfig{
			FanOnThreshold:     fanThresh,
			HeatingOnThreshold: heatThresh,
			CoolingOnThreshold: coolThresh,
		}
	}

	one := float32(1)

	tests := []struct {
		name     string
		cfg      *airTempModeConfig
		data     *modeDataPoints
		expected traits.AirTemperature_Mode
	}{
		{
			name:     "Cooling only, above threshold",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(nil, nil, float32Ptr(2)),
			expected: traits.AirTemperature_COOL,
		},
		{
			name:     "Cooling and Fan, both above threshold",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(2), nil, float32Ptr(2)),
			expected: traits.AirTemperature_COOL,
		},
		{
			name:     "Cooling and Fan, cooling above threshold, fan off",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(0), nil, float32Ptr(2)),
			expected: traits.AirTemperature_OFF,
		},
		{
			name:     "Heating only, above threshold",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(nil, float32Ptr(2), nil),
			expected: traits.AirTemperature_HEAT,
		},
		{
			name:     "Heating and Fan, both above threshold",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(2), float32Ptr(2), nil),
			expected: traits.AirTemperature_HEAT,
		},
		{
			name:     "Heating and Fan, heating above threshold, fan off",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(0), float32Ptr(2), nil),
			expected: traits.AirTemperature_OFF,
		},
		{
			name:     "Heating and Cooling, both above threshold, no Fan",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(nil, float32Ptr(2), float32Ptr(2)),
			expected: traits.AirTemperature_HEAT_COOL,
		},
		{
			name:     "Heating and Cooling, both above threshold, Fan above threshold",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(2), float32Ptr(2), float32Ptr(2)),
			expected: traits.AirTemperature_HEAT_COOL,
		},
		{
			name:     "Heating and Cooling, both above threshold, fan off",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(0), float32Ptr(2), float32Ptr(2)),
			expected: traits.AirTemperature_OFF,
		},
		{
			name:     "Fan only, above threshold",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(2), nil, nil),
			expected: traits.AirTemperature_FAN_ONLY,
		},
		{
			name:     "All below threshold",
			cfg:      newModeConfig(&one, &one, &one),
			data:     newModeDataPoints(float32Ptr(0), float32Ptr(0), float32Ptr(0)),
			expected: traits.AirTemperature_OFF,
		},
		{
			name: "Nil values everywhere",
			cfg:  newModeConfig(&one, &one, &one),
			data: newModeDataPoints(nil, nil, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			airTemp := &traits.AirTemperature{}
			updateMode(tt.cfg, tt.data, airTemp)
			assert.Equal(t, tt.expected, airTemp.Mode)
		})
	}
}

func TestReadAirTemperatureConfig_Defaults(t *testing.T) {
	raw := []byte(`{
		"name": "test"
	}`)
	cfg, err := readAirTemperatureConfig(raw)
	assert.NoError(t, err)
	assert.NotNil(t, cfg.SetPointDeadBand)
	assert.Equal(t, float32(1), *cfg.SetPointDeadBand)
}

func TestReadAirTemperatureConfig_ModeConfigDefaults(t *testing.T) {
	raw := []byte(`{
		"name": "test",
		"modeConfig": {}
	}`)
	cfg, err := readAirTemperatureConfig(raw)
	assert.NoError(t, err)
	assert.NotNil(t, cfg.ModeConfig)
	assert.NotNil(t, cfg.ModeConfig.FanOnThreshold)
	assert.NotNil(t, cfg.ModeConfig.HeatingOnThreshold)
	assert.NotNil(t, cfg.ModeConfig.CoolingOnThreshold)
	assert.Equal(t, float32(1), *cfg.ModeConfig.FanOnThreshold)
	assert.Equal(t, float32(1), *cfg.ModeConfig.HeatingOnThreshold)
	assert.Equal(t, float32(1), *cfg.ModeConfig.CoolingOnThreshold)
}

func TestGetModePoints_Combinations(t *testing.T) {
	vs := func(name int) *config.ValueSource {
		pid := config.PropertyID(name)
		return &config.ValueSource{Property: &pid}
	}

	type testCase struct {
		name           string
		modeConfig     *airTempModeConfig
		expectedNames  []string
		expectedValues []config.ValueSource
	}

	tests := []testCase{
		{
			name: "All nil",
			modeConfig: &airTempModeConfig{
				FanOn:     nil,
				HeatingOn: nil,
				CoolingOn: nil,
			},
			expectedNames:  nil,
			expectedValues: nil,
		},
		{
			name: "FanOn only",
			modeConfig: &airTempModeConfig{
				FanOn:     vs(1),
				HeatingOn: nil,
				CoolingOn: nil,
			},
			expectedNames:  []string{"fanOn"},
			expectedValues: []config.ValueSource{*vs(1)},
		},
		{
			name: "HeatingOn only",
			modeConfig: &airTempModeConfig{
				FanOn:     nil,
				HeatingOn: vs(2),
				CoolingOn: nil,
			},
			expectedNames:  []string{"heatingOn"},
			expectedValues: []config.ValueSource{*vs(2)},
		},
		{
			name: "CoolingOn only",
			modeConfig: &airTempModeConfig{
				FanOn:     nil,
				HeatingOn: nil,
				CoolingOn: vs(3),
			},
			expectedNames:  []string{"coolingOn"},
			expectedValues: []config.ValueSource{*vs(3)},
		},
		{
			name: "FanOn and HeatingOn",
			modeConfig: &airTempModeConfig{
				FanOn:     vs(1),
				HeatingOn: vs(2),
				CoolingOn: nil,
			},
			expectedNames:  []string{"fanOn", "heatingOn"},
			expectedValues: []config.ValueSource{*vs(1), *vs(2)},
		},
		{
			name: "FanOn and CoolingOn",
			modeConfig: &airTempModeConfig{
				FanOn:     vs(1),
				HeatingOn: nil,
				CoolingOn: vs(3),
			},
			expectedNames:  []string{"fanOn", "coolingOn"},
			expectedValues: []config.ValueSource{*vs(1), *vs(3)},
		},
		{
			name: "HeatingOn and CoolingOn",
			modeConfig: &airTempModeConfig{
				FanOn:     nil,
				HeatingOn: vs(2),
				CoolingOn: vs(3),
			},
			expectedNames:  []string{"heatingOn", "coolingOn"},
			expectedValues: []config.ValueSource{*vs(2), *vs(3)},
		},
		{
			name: "All present",
			modeConfig: &airTempModeConfig{
				FanOn:     vs(1),
				HeatingOn: vs(2),
				CoolingOn: vs(3),
			},
			expectedNames:  []string{"fanOn", "heatingOn", "coolingOn"},
			expectedValues: []config.ValueSource{*vs(1), *vs(2), *vs(3)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tgt := &airTemperature{
				config: airTemperatureConfig{
					ModeConfig: tc.modeConfig,
				},
			}
			var procs []func(response any) error
			var vals []config.ValueSource
			var names []string
			_ = tgt.getModePoints(&procs, &vals, &names)
			assert.Equal(t, tc.expectedNames, names)
			assert.Equal(t, tc.expectedValues, vals)
			assert.Equal(t, len(tc.expectedNames), len(procs))
		})
	}
}
