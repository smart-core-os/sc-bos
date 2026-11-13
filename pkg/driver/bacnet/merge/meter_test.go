package merge

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

func newTestMeter(t *testing.T, raw string) (*meterTrait, *observer.ObservedLogs) {
	t.Helper()
	var cfg config.RawTrait
	if err := cfg.UnmarshalJSON([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	core, logs := observer.New(zapcore.WarnLevel)
	m, err := newMeter(nil, nil, nil, cfg, zap.New(core))
	if err != nil {
		t.Fatal(err)
	}
	m.readUnits = func(context.Context) (any, error) {
		// not t.Fatal: this runs on the once.RetryError goroutine
		t.Error("readUnits called unexpectedly")
		return nil, errors.New("unexpected read")
	}
	return m, logs
}

func describeUnit(t *testing.T, m *meterTrait) string {
	t.Helper()
	res, err := m.DescribeMeterReading(t.Context(), &meterpb.DescribeMeterReadingRequest{Name: "meter"})
	if err != nil {
		t.Fatalf("DescribeMeterReading: %v", err)
	}
	return res.UsageUnit
}

const meterUsageCfg = `{"name":"meter","kind":"Meter","usage":{"device":1,"object":"AnalogInput:1"}`

func TestMeterTrait_DescribeMeterReading(t *testing.T) {
	t.Run("config unit wins", func(t *testing.T) {
		m, _ := newTestMeter(t, meterUsageCfg+`,"unit":"kWh"}`)
		if got := describeUnit(t, m); got != "kWh" {
			t.Errorf("unit = %q, want kWh", got)
		}
	})

	t.Run("units from device", func(t *testing.T) {
		tests := []struct {
			units any
			want  string
		}{
			{uint32(18), "Wh"},
			{uint32(19), "kWh"},
			{uint32(146), "MWh"},
			{uint32(80), "m³"},
			{uint32(82), "L"},
			{uint32(47), "W"}, // not a meter unit, falls back to the generic symbol
		}
		for _, tt := range tests {
			m, logs := newTestMeter(t, meterUsageCfg+`}`)
			var calls int
			m.readUnits = func(context.Context) (any, error) {
				calls++
				return tt.units, nil
			}
			for range 2 {
				if got := describeUnit(t, m); got != tt.want {
					t.Errorf("units %v: unit = %q, want %q", tt.units, got, tt.want)
				}
			}
			if calls != 1 {
				t.Errorf("units %v: read units %d times, want 1", tt.units, calls)
			}
			if logs.Len() != 0 {
				t.Errorf("units %v: unexpected warnings %v", tt.units, logs.All())
			}
		}
	})

	t.Run("read failure retries", func(t *testing.T) {
		m, logs := newTestMeter(t, meterUsageCfg+`}`)
		var calls int
		m.readUnits = func(context.Context) (any, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("device offline")
			}
			return uint32(19), nil
		}
		// an error, not "": consumers take "" as the meter's answer and stop asking
		_, err := m.DescribeMeterReading(t.Context(), &meterpb.DescribeMeterReadingRequest{Name: "meter"})
		if status.Code(err) != codes.Unavailable {
			t.Errorf("first describe err = %v, want Unavailable", err)
		}
		if logs.FilterMessageSnippet("reading meter units").Len() != 1 {
			t.Errorf("want a warning about the failed read, got %v", logs.All())
		}
		if got := describeUnit(t, m); got != "kWh" {
			t.Errorf("second unit = %q, want kWh", got)
		}
		if calls != 2 {
			t.Errorf("read units %d times, want 2", calls)
		}
	})

	t.Run("no units on device", func(t *testing.T) {
		m, logs := newTestMeter(t, meterUsageCfg+`}`)
		m.readUnits = func(context.Context) (any, error) { return uint32(95), nil } // no-units
		for range 3 {
			if got := describeUnit(t, m); got != "" {
				t.Errorf("unit = %q, want empty", got)
			}
		}
		if n := logs.FilterMessageSnippet("no usable unit").Len(); n != 1 {
			t.Errorf("want one no-unit warning, got %d: %v", n, logs.All())
		}
	})

	t.Run("scaled usage without unit", func(t *testing.T) {
		// readUnits fails the test if called: the device's units describe the unscaled value
		m, logs := newTestMeter(t, `{"name":"meter","kind":"Meter","usage":{"device":1,"object":"AnalogInput:1","scale":0.001}}`)
		for range 2 {
			if got := describeUnit(t, m); got != "" {
				t.Errorf("unit = %q, want empty", got)
			}
		}
		if n := logs.FilterMessageSnippet("scaled").Len(); n != 1 {
			t.Errorf("want one scaled warning, got %d: %v", n, logs.All())
		}
	})

	t.Run("scaled usage with unit", func(t *testing.T) {
		m, _ := newTestMeter(t, `{"name":"meter","kind":"Meter","unit":"kWh","usage":{"device":1,"object":"AnalogInput:1","scale":0.001}}`)
		if got := describeUnit(t, m); got != "kWh" {
			t.Errorf("unit = %q, want kWh", got)
		}
	})
}
