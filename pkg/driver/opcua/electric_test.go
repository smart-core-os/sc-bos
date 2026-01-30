package opcua

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
)

func TestElectric_GetDemand(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`),
	}

	electric, err := newElectric("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newElectric() error = %v", err)
	}

	demand, err := electric.GetDemand(t.Context(), &traits.GetDemandRequest{})
	if err != nil {
		t.Errorf("GetDemand() error = %v", err)
	}
	if demand == nil {
		t.Fatal("GetDemand() returned nil")
	}
}

func TestElectric_handleElectricEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      string
		nodeId      string
		value       any
		checkField  string
		wantValue   float32
		wantHandled bool
	}{
		{
			name:        "real power",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=Power",
			value:       float32(1500.5),
			checkField:  "realPower",
			wantValue:   1500.5,
			wantHandled: true,
		},
		{
			name:        "apparent power",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"apparentPower":{"nodeId":"ns=2;s=ApparentPower"}}}`,
			nodeId:      "ns=2;s=ApparentPower",
			value:       float32(2000.0),
			checkField:  "apparentPower",
			wantValue:   2000.0,
			wantHandled: true,
		},
		{
			name:        "reactive power",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"reactivePower":{"nodeId":"ns=2;s=ReactivePower"}}}`,
			nodeId:      "ns=2;s=ReactivePower",
			value:       float32(500.0),
			checkField:  "reactivePower",
			wantValue:   500.0,
			wantHandled: true,
		},
		{
			name:        "power factor",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"powerFactor":{"nodeId":"ns=2;s=PF"}}}`,
			nodeId:      "ns=2;s=PF",
			value:       float32(0.95),
			checkField:  "powerFactor",
			wantValue:   0.95,
			wantHandled: true,
		},
		{
			name:        "real power with scaling (kW to W)",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power","scale":1000}}}`,
			nodeId:      "ns=2;s=Power",
			value:       float32(2.5),
			checkField:  "realPower",
			wantValue:   2500.0,
			wantHandled: true,
		},
		{
			name:        "int32 value converted to float32",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=Power",
			value:       int32(1000),
			checkField:  "realPower",
			wantValue:   1000.0,
			wantHandled: true,
		},
		{
			name:        "wrong node id - not handled",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=WrongNode",
			value:       float32(1500.0),
			checkField:  "realPower",
			wantValue:   0,
			wantHandled: false,
		},
		{
			name:        "invalid value type",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=Power",
			value:       "not a number",
			checkField:  "realPower",
			wantValue:   0,
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			electric, err := newElectric("test/device", cfg, logger)
			if err != nil {
				t.Fatalf("newElectric() error = %v", err)
			}

			nodeId, _ := ua.ParseNodeID(tt.nodeId)
			electric.handleEvent(t.Context(), nodeId, tt.value)

			demand, _ := electric.GetDemand(t.Context(), &traits.GetDemandRequest{})

			var actualValue float32
			switch tt.checkField {
			case "realPower":
				if demand.RealPower != nil {
					actualValue = *demand.RealPower
				}
			case "apparentPower":
				if demand.ApparentPower != nil {
					actualValue = *demand.ApparentPower
				}
			case "reactivePower":
				if demand.ReactivePower != nil {
					actualValue = *demand.ReactivePower
				}
			case "powerFactor":
				if demand.PowerFactor != nil {
					actualValue = *demand.PowerFactor
				}
			}

			if actualValue != tt.wantValue {
				t.Errorf("After handleElectricEvent, %s = %v, want %v", tt.checkField, actualValue, tt.wantValue)
			}
		})
	}
}

func TestElectric_handleElectricEvent_NoDemand(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create Electric with nil demand (should be caught by validation, but test defensive code)
	electric := &Electric{
		cfg:    config.ElectricConfig{},
		logger: logger,
		scName: "test/device",
	}

	nodeId, _ := ua.ParseNodeID("ns=2;s=Power")
	// Should not panic, just log warning and return
	electric.handleEvent(t.Context(), nodeId, float32(100.0))
}

func TestElectric_newElectric_InvalidConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name:      "invalid json",
			config:    `{invalid json}`,
			wantError: true,
		},
		{
			name:      "valid config",
			config:    `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			_, err := newElectric("test/device", cfg, logger)
			if (err != nil) != tt.wantError {
				t.Errorf("newElectric() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
