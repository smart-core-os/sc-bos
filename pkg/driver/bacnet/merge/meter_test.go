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

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
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
	unexpected := func(context.Context) (any, error) {
		// not t.Fatal: this runs on the once.RetryError goroutine
		t.Error("units read unexpectedly")
		return nil, errors.New("unexpected read")
	}
	m.usageUnit.read = unexpected
	if m.producedUnit != nil {
		m.producedUnit.read = unexpected
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
			m.usageUnit.read = func(context.Context) (any, error) {
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
		m.usageUnit.read = func(context.Context) (any, error) {
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
		m.usageUnit.read = func(context.Context) (any, error) { return uint32(95), nil } // no-units
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

func TestReadMeterConfig(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		wantUsage      bool
		wantProduction bool
		wantErr        bool
	}{
		{name: "usage only", raw: `{"usage":{"object":"AnalogValue:1"}}`, wantUsage: true},
		{name: "usage and production", raw: `{"usage":{"object":"AnalogValue:1"},"production":{"object":"AnalogValue:2"}}`, wantUsage: true, wantProduction: true},
		{name: "production only", raw: `{"production":{"object":"AnalogValue:2"}}`, wantProduction: true},
		// a meter that reads nothing used to nil-panic on its first poll
		{name: "neither", raw: `{"unit":"kWh"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := readMeterConfig([]byte(tt.raw))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got := cfg.Usage != nil; got != tt.wantUsage {
				t.Errorf("usage parsed = %v, want %v", got, tt.wantUsage)
			}
			if got := cfg.Production != nil; got != tt.wantProduction {
				t.Errorf("production parsed = %v, want %v", got, tt.wantProduction)
			}
		})
	}
}

func TestMeterTrait_DescribeMeterReading_producedUnit(t *testing.T) {
	const production = `"production":{"device":1,"object":"AnalogInput:2"`
	describe := func(t *testing.T, m *meterTrait) *meterpb.MeterReadingSupport {
		t.Helper()
		res, err := m.DescribeMeterReading(t.Context(), &meterpb.DescribeMeterReadingRequest{Name: "meter"})
		if err != nil {
			t.Fatalf("DescribeMeterReading: %v", err)
		}
		return res
	}
	// unitsFrom returns a units read that reports u.
	unitsFrom := func(u uint32) func(context.Context) (any, error) {
		return func(context.Context) (any, error) { return u, nil }
	}
	assertUnits := func(t *testing.T, res *meterpb.MeterReadingSupport, usage, produced string) {
		t.Helper()
		if res.UsageUnit != usage || res.ProducedUnit != produced {
			t.Errorf("units = (%q, %q), want (%q, %q)", res.UsageUnit, res.ProducedUnit, usage, produced)
		}
	}

	t.Run("usage only leaves produced unit empty", func(t *testing.T) {
		// an empty produced unit tells consumers this meter reports no export
		m, _ := newTestMeter(t, meterUsageCfg+`,"unit":"kWh"}`)
		if m.producedUnit != nil {
			t.Fatal("produced unit resolver exists without production configured")
		}
		assertUnits(t, describe(t, m), "kWh", "")
	})

	t.Run("each reading takes its own configured unit", func(t *testing.T) {
		m, _ := newTestMeter(t, meterUsageCfg+`,"unit":"kWh","producedUnit":"Wh",`+production+`}}`)
		assertUnits(t, describe(t, m), "kWh", "Wh")
	})

	t.Run("unit does not cover production", func(t *testing.T) {
		// the production object reports Wh; taking the usage unit would misreport it by 1000x
		m, _ := newTestMeter(t, meterUsageCfg+`,"unit":"kWh",`+production+`}}`)
		m.producedUnit.read = unitsFrom(18) // Wh
		assertUnits(t, describe(t, m), "kWh", "Wh")
	})

	t.Run("units read from each object", func(t *testing.T) {
		m, _ := newTestMeter(t, meterUsageCfg+`,`+production+`}}`)
		m.usageUnit.read = unitsFrom(19)    // kWh
		m.producedUnit.read = unitsFrom(18) // Wh
		assertUnits(t, describe(t, m), "kWh", "Wh")
	})

	t.Run("production only", func(t *testing.T) {
		m, _ := newTestMeter(t, `{"name":"meter","kind":"Meter",`+production+`}}`)
		m.producedUnit.read = unitsFrom(19) // kWh
		assertUnits(t, describe(t, m), "", "kWh")
	})

	t.Run("scaled production without producedUnit", func(t *testing.T) {
		// the device's units describe the unscaled value, so neither they nor unit are used
		m, logs := newTestMeter(t, meterUsageCfg+`,"unit":"kWh","production":{"device":1,"object":"AnalogInput:2","scale":0.001}}`)
		assertUnits(t, describe(t, m), "kWh", "")
		if n := logs.FilterMessageSnippet("set producedUnit").Len(); n != 1 {
			t.Errorf("want one warning naming producedUnit, got %d: %v", n, logs.All())
		}
	})

	t.Run("production units unreadable", func(t *testing.T) {
		m, _ := newTestMeter(t, meterUsageCfg+`,"unit":"kWh",`+production+`}}`)
		m.producedUnit.read = func(context.Context) (any, error) { return nil, errors.New("device offline") }
		// an error, not "": "" would tell consumers the meter reports no export
		_, err := m.DescribeMeterReading(t.Context(), &meterpb.DescribeMeterReadingRequest{Name: "meter"})
		if status.Code(err) != codes.Unavailable {
			t.Errorf("err = %v, want Unavailable", err)
		}
	})

	t.Run("units are read from the reading's own object", func(t *testing.T) {
		m, _ := newTestMeter(t, meterUsageCfg+`,`+production+`}}`)
		if m.usageUnit.src != m.config.Usage || m.producedUnit.src != m.config.Production {
			t.Error("a reading's unit resolver points at the wrong object")
		}
	})
}

func TestApplyMeterReads(t *testing.T) {
	m, _ := newTestMeter(t, meterUsageCfg+`,"production":{"device":1,"object":"AnalogInput:2"}}`)
	reads := m.meterReads()
	if len(reads) != 2 || reads[0].prop != "usage" || reads[1].prop != "production" {
		t.Fatalf("reads = %v, want usage then production", reads)
	}

	t.Run("each response goes to its own field", func(t *testing.T) {
		data, errs := applyMeterReads(reads, []any{float32(12.5), float32(3.25)})
		if len(errs) != 0 {
			t.Fatalf("errs = %v", errs)
		}
		if data.Usage != 12.5 || data.Produced != 3.25 {
			t.Errorf("reading = (usage %v, produced %v), want (12.5, 3.25)", data.Usage, data.Produced)
		}
	})

	t.Run("a failed read is reported against its own reading", func(t *testing.T) {
		data, errs := applyMeterReads(reads, []any{float32(12.5), errors.New("no response")})
		if len(errs) != 1 {
			t.Fatalf("errs = %v, want one", errs)
		}
		var rp comm.ErrReadProperty
		if !errors.As(errs[0], &rp) || rp.Prop != "production" {
			t.Errorf("err = %v, want a production read error", errs[0])
		}
		if data.Usage != 12.5 {
			t.Errorf("usage = %v, want 12.5", data.Usage)
		}
	})

	t.Run("production only", func(t *testing.T) {
		m, _ := newTestMeter(t, `{"name":"meter","kind":"Meter","production":{"device":1,"object":"AnalogInput:2"}}`)
		data, errs := applyMeterReads(m.meterReads(), []any{float32(7)})
		if len(errs) != 0 || data.Produced != 7 || data.Usage != 0 {
			t.Errorf("reading = (usage %v, produced %v), errs %v; want (0, 7), none", data.Usage, data.Produced, errs)
		}
	})
}
