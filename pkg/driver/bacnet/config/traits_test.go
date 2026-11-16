package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/smart-core-os/gobacnet/types/objecttype"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func TestObjectRef_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		want    ObjectRef
		wantErr bool
	}{
		{`""`, ObjectRef{}, false},
		{`"some string"`, ObjectRef{name: "some string"}, false},
		{`"AnalogValue:12"`, ObjectRef{id: ObjectID{Type: objecttype.AnalogValue, Instance: 12}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ObjectRef
			if err := (&got).UnmarshalJSON([]byte(tt.name)); (err != nil) != tt.wantErr {
				t.Errorf("ObjectRef.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ObjectRef.UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTraitHealth_FaultAfterDuration(t *testing.T) {
	tests := []struct {
		name string
		json string
		want time.Duration
	}{
		{"no health", `{"name":"t"}`, DefaultTraitFaultAfter},
		{"health without faultAfter", `{"health":{"occupantImpact":"COMFORT"}}`, DefaultTraitFaultAfter},
		{"faultAfter", `{"health":{"faultAfter":"30s"}}`, 30 * time.Second},
		{"zero faultAfter", `{"health":{"faultAfter":"0s"}}`, 0},
		{"negative faultAfter", `{"health":{"faultAfter":"-1m"}}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got RawTrait
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if d := got.Health.FaultAfterDuration(); d != tt.want {
				t.Errorf("FaultAfterDuration() = %v, want %v", d, tt.want)
			}
		})
	}
}

// TestTraitHealth_EmbedsHealth verifies the impacts still parse alongside faultAfter.
func TestTraitHealth_EmbedsHealth(t *testing.T) {
	var got RawTrait
	if err := json.Unmarshal([]byte(`{"health":{"occupantImpact":"COMFORT","equipmentImpact":"FUNCTION","faultAfter":"1m"}}`), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Health.OccupantImpact.ToProto() != healthpb.HealthCheck_COMFORT {
		t.Errorf("OccupantImpact = %v, want COMFORT", got.Health.OccupantImpact.ToProto())
	}
	if got.Health.EquipmentImpact.ToProto() != healthpb.HealthCheck_FUNCTION {
		t.Errorf("EquipmentImpact = %v, want FUNCTION", got.Health.EquipmentImpact.ToProto())
	}
}

func TestValueSource_Unscaled(t *testing.T) {
	tests := []struct {
		name  string
		scale float64
		in    any
		want  any
	}{
		{"no scale float32", 0, float32(2.5), float32(2.5)},
		{"no scale int", 0, 7, 7},
		{"nil", 1000, nil, nil},
		{"not a number", 1000, "on", "on"},
		{"bool", 1000, true, true},
		{"float32 kilo", 1000, float32(2500), float32(2.5)},
		{"float64 kilo", 1000, float64(2500), float64(2.5)},
		{"float64 tenth", 0.1, float64(21.5), float64(215)},
		{"int rounds up", 3, 14, 5},          // 4.67
		{"int rounds down", 3, 13, 4},        // 4.33
		{"int32", 10, int32(255), int32(26)}, // 25.5 rounds away from zero
		{"int64", 0.5, int64(4), int64(8)},
		{"uint", 2, uint(9), uint(5)},
		{"uint32", 0.1, uint32(3), uint32(30)},
		{"uint64", 100, uint64(149), uint64(1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := ValueSource{Scale: tt.scale}
			if got := vs.Unscaled(tt.in); got != tt.want {
				t.Errorf("Unscaled(%v) = %v (%T), want %v (%T)", tt.in, got, got, tt.want, tt.want)
			}
		})
	}
}

// Writes must land on the device in its own units, so a value read through Scaled and written back through Unscaled
// should come back unchanged.
func TestValueSource_ScaledUnscaledRoundTrip(t *testing.T) {
	scales := []float64{0, 1, 1000, 0.001, 0.1, 3.6}
	values := []any{float32(21.5), float64(1234.5678), 42, int32(-7), int64(1 << 40), uint(12), uint32(250), uint64(99)}
	for _, scale := range scales {
		vs := ValueSource{Scale: scale}
		for _, v := range values {
			got := vs.Scaled(vs.Unscaled(v))
			switch want := v.(type) {
			case float32:
				g := got.(float32)
				if diff := float64(g - want); diff > 1e-3 || diff < -1e-3 {
					t.Errorf("scale %v: Scaled(Unscaled(%v)) = %v", scale, v, got)
				}
			case float64:
				g := got.(float64)
				if diff := g - want; diff > 1e-9 || diff < -1e-9 {
					t.Errorf("scale %v: Scaled(Unscaled(%v)) = %v", scale, v, got)
				}
			default:
				// Integers can't round-trip exactly when the scale loses precision (e.g. 42/1000 rounds to 0),
				// so only check the scales that keep whole numbers whole.
				if scale == 0 || scale == 1 || scale == 0.1 || scale == 0.001 {
					if got != v {
						t.Errorf("scale %v: Scaled(Unscaled(%v)) = %v (%T)", scale, v, got, got)
					}
				}
			}
		}
	}
}
