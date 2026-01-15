package opcua

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

func TestMeter_GetMeterReading(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
	}

	meter, err := newMeter("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newMeter() error = %v", err)
	}

	// Test initial state
	reading, err := meter.GetMeterReading(t.Context(), &meterpb.GetMeterReadingRequest{})
	if err != nil {
		t.Errorf("GetMeterReading() error = %v", err)
	}
	if reading == nil {
		t.Fatal("GetMeterReading() returned nil")
	}
	if reading.Usage != 0 {
		t.Errorf("Initial usage = %v, want 0", reading.Usage)
	}
}

func TestMeter_DescribeMeterReading(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
	}

	meter, err := newMeter("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newMeter() error = %v", err)
	}

	support, err := meter.DescribeMeterReading(t.Context(), &meterpb.DescribeMeterReadingRequest{})
	if err != nil {
		t.Errorf("DescribeMeterReading() error = %v", err)
	}
	if support.UsageUnit != "kWh" {
		t.Errorf("UsageUnit = %v, want kWh", support.UsageUnit)
	}
}

func TestMeter_handleMeterEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      string
		nodeId      string
		value       any
		wantUsage   float32
		wantHandled bool
	}{
		{
			name:        "valid float32 value",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       float32(123.45),
			wantUsage:   123.45,
			wantHandled: true,
		},
		{
			name:        "valid int32 value",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       int32(100),
			wantUsage:   100.0,
			wantHandled: true,
		},
		{
			name:        "value with scaling",
			config:      `{"kind":"smartcore.bos.Meter","unit":"Wh","usage":{"nodeId":"ns=2;s=Tag1","scale":1000}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       float32(2.5),
			wantUsage:   2500.0,
			wantHandled: true,
		},
		{
			name:        "wrong node id - not handled",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag2",
			value:       float32(123.45),
			wantUsage:   0,
			wantHandled: false,
		},
		{
			name:        "invalid value type - not handled",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       "not a number",
			wantUsage:   0,
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			meter, err := newMeter("test/device", cfg, logger)
			if err != nil {
				t.Fatalf("newMeter() error = %v", err)
			}

			nodeId, _ := ua.ParseNodeID(tt.nodeId)
			meter.handleMeterEvent(nodeId, tt.value)

			reading, _ := meter.GetMeterReading(t.Context(), &meterpb.GetMeterReadingRequest{})
			if reading.Usage != tt.wantUsage {
				t.Errorf("After handleMeterEvent, usage = %v, want %v", reading.Usage, tt.wantUsage)
			}
		})
	}
}

func TestMeter_newMeter_InvalidConfig(t *testing.T) {
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
			config:    `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			_, err := newMeter("test/device", cfg, logger)
			if (err != nil) != tt.wantError {
				t.Errorf("newMeter() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
