package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func TestOccupantImpact_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    healthpb.HealthCheck_OccupantImpact
		wantErr bool
	}{
		{
			name: "OCCUPANT_IMPACT_UNSPECIFIED",
			json: `"OCCUPANT_IMPACT_UNSPECIFIED"`,
			want: healthpb.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED,
		},
		{
			name: "NO_OCCUPANT_IMPACT",
			json: `"NO_OCCUPANT_IMPACT"`,
			want: healthpb.HealthCheck_NO_OCCUPANT_IMPACT,
		},
		{
			name: "COMFORT lowercase",
			json: `"comfort"`,
			want: healthpb.HealthCheck_COMFORT,
		},
		{
			name: "HEALTH",
			json: `"HEALTH"`,
			want: healthpb.HealthCheck_HEALTH,
		},
		{
			name: "LIFE",
			json: `"LIFE"`,
			want: healthpb.HealthCheck_LIFE,
		},
		{
			name: "SECURITY",
			json: `"SECURITY"`,
			want: healthpb.HealthCheck_SECURITY,
		},
		{
			name:    "invalid value",
			json:    `"INVALID"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o OccupantImpact
			err := json.Unmarshal([]byte(tt.json), &o)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, o.ToProto())
		})
	}
}

func TestEquipmentImpact_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    healthpb.HealthCheck_EquipmentImpact
		wantErr bool
	}{
		{
			name: "EQUIPMENT_IMPACT_UNSPECIFIED",
			json: `"EQUIPMENT_IMPACT_UNSPECIFIED"`,
			want: healthpb.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED,
		},
		{
			name: "NO_EQUIPMENT_IMPACT",
			json: `"NO_EQUIPMENT_IMPACT"`,
			want: healthpb.HealthCheck_NO_EQUIPMENT_IMPACT,
		},
		{
			name: "WARRANTY lowercase",
			json: `"warranty"`,
			want: healthpb.HealthCheck_WARRANTY,
		},
		{
			name: "LIFESPAN",
			json: `"LIFESPAN"`,
			want: healthpb.HealthCheck_LIFESPAN,
		},
		{
			name: "FUNCTION",
			json: `"FUNCTION"`,
			want: healthpb.HealthCheck_FUNCTION,
		},
		{
			name:    "invalid value",
			json:    `"INVALID"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e EquipmentImpact
			err := json.Unmarshal([]byte(tt.json), &e)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, e.ToProto())
		})
	}
}

func TestHealth_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"occupantImpact": "COMFORT",
		"equipmentImpact": "WARRANTY"
	}`

	var h Health
	err := json.Unmarshal([]byte(jsonData), &h)
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheck_COMFORT, h.OccupantImpact.ToProto())
	require.Equal(t, healthpb.HealthCheck_WARRANTY, h.EquipmentImpact.ToProto())
}
