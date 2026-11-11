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
