package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smart-core-os/sc-bos/pkg/gen"
)

func TestOccupantImpact_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    gen.HealthCheck_OccupantImpact
		wantErr bool
	}{
		{
			name: "OCCUPANT_IMPACT_UNSPECIFIED",
			json: `"OCCUPANT_IMPACT_UNSPECIFIED"`,
			want: gen.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED,
		},
		{
			name: "NO_OCCUPANT_IMPACT",
			json: `"NO_OCCUPANT_IMPACT"`,
			want: gen.HealthCheck_NO_OCCUPANT_IMPACT,
		},
		{
			name: "COMFORT lowercase",
			json: `"comfort"`,
			want: gen.HealthCheck_COMFORT,
		},
		{
			name: "HEALTH",
			json: `"HEALTH"`,
			want: gen.HealthCheck_HEALTH,
		},
		{
			name: "LIFE",
			json: `"LIFE"`,
			want: gen.HealthCheck_LIFE,
		},
		{
			name: "SECURITY",
			json: `"SECURITY"`,
			want: gen.HealthCheck_SECURITY,
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
		want    gen.HealthCheck_EquipmentImpact
		wantErr bool
	}{
		{
			name: "EQUIPMENT_IMPACT_UNSPECIFIED",
			json: `"EQUIPMENT_IMPACT_UNSPECIFIED"`,
			want: gen.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED,
		},
		{
			name: "NO_EQUIPMENT_IMPACT",
			json: `"NO_EQUIPMENT_IMPACT"`,
			want: gen.HealthCheck_NO_EQUIPMENT_IMPACT,
		},
		{
			name: "WARRANTY lowercase",
			json: `"warranty"`,
			want: gen.HealthCheck_WARRANTY,
		},
		{
			name: "LIFESPAN",
			json: `"LIFESPAN"`,
			want: gen.HealthCheck_LIFESPAN,
		},
		{
			name: "FUNCTION",
			json: `"FUNCTION"`,
			want: gen.HealthCheck_FUNCTION,
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
		"equipmentImpact": "WARRANTY",
		"systemName": "test-system"
	}`

	var h Health
	err := json.Unmarshal([]byte(jsonData), &h)
	require.NoError(t, err)
	require.Equal(t, gen.HealthCheck_COMFORT, h.OccupantImpact.ToProto())
	require.Equal(t, gen.HealthCheck_WARRANTY, h.EquipmentImpact.ToProto())
	require.Equal(t, "test-system", h.SystemName)
}
