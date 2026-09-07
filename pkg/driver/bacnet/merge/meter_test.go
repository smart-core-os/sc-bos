package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

func TestReadMeterConfig(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		wantUsage      bool
		wantProduction bool
		wantUnit       string
	}{
		{
			name:      "usage only",
			raw:       `{"name": "test", "unit": "kWh", "usage": {"object": "analog-value,1"}}`,
			wantUsage: true,
			wantUnit:  "kWh",
		},
		{
			name:           "usage and production",
			raw:            `{"name": "test", "unit": "kWh", "usage": {"object": "analog-value,1"}, "production": {"object": "analog-value,2"}}`,
			wantUsage:      true,
			wantProduction: true,
			wantUnit:       "kWh",
		},
		{
			name:           "production only",
			raw:            `{"name": "test", "unit": "kWh", "production": {"object": "analog-value,2"}}`,
			wantProduction: true,
			wantUnit:       "kWh",
		},
		{
			name: "neither",
			raw:  `{"name": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := readMeterConfig([]byte(tt.raw))
			assert.NoError(t, err)
			assert.Equal(t, tt.wantUnit, cfg.Unit)
			if tt.wantUsage {
				assert.NotNil(t, cfg.Usage, "usage should be parsed")
			} else {
				assert.Nil(t, cfg.Usage, "usage should be absent")
			}
			if tt.wantProduction {
				assert.NotNil(t, cfg.Production, "production should be parsed")
			} else {
				assert.Nil(t, cfg.Production, "production should be absent")
			}
		})
	}
}

func TestMeterReadingSupport_ProducedUnit(t *testing.T) {
	vs := func() *config.ValueSource {
		pid := config.PropertyID(85)
		return &config.ValueSource{Property: &pid}
	}

	tests := []struct {
		name             string
		cfg              meterConfig
		wantUsageUnit    string
		wantProducedUnit string
	}{
		{
			name:          "usage only leaves produced unit empty",
			cfg:           meterConfig{Unit: "kWh", Usage: vs()},
			wantUsageUnit: "kWh",
			// empty produced unit tells consumers this meter reports no export
			wantProducedUnit: "",
		},
		{
			name:             "production shares the configured unit",
			cfg:              meterConfig{Unit: "kWh", Usage: vs(), Production: vs()},
			wantUsageUnit:    "kWh",
			wantProducedUnit: "kWh",
		},
		{
			name:             "production only still declares both units",
			cfg:              meterConfig{Unit: "kWh", Production: vs()},
			wantUsageUnit:    "kWh",
			wantProducedUnit: "kWh",
		},
		{
			name: "no unit configured",
			cfg:  meterConfig{Production: vs()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			support := meterReadingSupport(tt.cfg)
			assert.Equal(t, tt.wantUsageUnit, support.UsageUnit)
			assert.Equal(t, tt.wantProducedUnit, support.ProducedUnit)
			assert.True(t, support.ResourceSupport.Readable)
			assert.True(t, support.ResourceSupport.Observable)
		})
	}
}
