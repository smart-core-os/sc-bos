package connecttelemetry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

func boundsCheck(id string, value float64, normality healthpb.HealthCheck_Normality) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:        id,
		Normality: normality,
		Deviation: value / 10,
		Check: &healthpb.HealthCheck_Bounds_{Bounds: &healthpb.HealthCheck_Bounds{
			CurrentValue: healthpb.FloatValue(value),
			Expected: &healthpb.HealthCheck_Bounds_NormalRange{NormalRange: &healthpb.HealthCheck_ValueRange{
				Low:  healthpb.FloatValue(0),
				High: healthpb.FloatValue(100),
			}},
		}},
	}
}

func TestCheckSetHash(t *testing.T) {
	a := boundsCheck("a", 20, healthpb.HealthCheck_NORMAL)
	b := &healthpb.HealthCheck{Id: "b", DisplayName: "b check"}
	base := checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{a, b})
	assert.Len(t, base, 16)

	t.Run("ignores measured values", func(t *testing.T) {
		moved := boundsCheck("a", 40, healthpb.HealthCheck_NORMAL)
		assert.Equal(t, base, checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{moved, b}))
	})
	t.Run("does not modify checks", func(t *testing.T) {
		assert.Equal(t, 20.0, a.GetBounds().GetCurrentValue().GetFloatValue())
		assert.NotZero(t, a.Deviation)
	})
	t.Run("ignores order", func(t *testing.T) {
		assert.Equal(t, base, checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{b, a}))
	})
	t.Run("changes with normality", func(t *testing.T) {
		high := boundsCheck("a", 120, healthpb.HealthCheck_HIGH)
		assert.NotEqual(t, base, checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{high, b}))
	})
	t.Run("changes with kind", func(t *testing.T) {
		assert.NotEqual(t, base, checkSetHash(subjectKindService, []*healthpb.HealthCheck{a, b}))
	})
	t.Run("changes with membership", func(t *testing.T) {
		assert.NotEqual(t, base, checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{a}))
		assert.NotEqual(t, checkSetHash(subjectKindDevice, nil), checkSetHash(subjectKindDevice, []*healthpb.HealthCheck{a}))
	})
}

func TestNameHash(t *testing.T) {
	// the worked example in Connect's docs/health.md
	assert.Equal(t, "f5b04fb7c61623cf", nameHash("pier-point/hvac/ctrl-3"))
}

func TestBuildHealthSet(t *testing.T) {
	now := time.Date(2026, 10, 13, 9, 12, 4, 0, time.FixedZone("BST", 3600))

	t.Run("empty set", func(t *testing.T) {
		b, err := buildHealthSet(now, "pier-point/hvac/ctrl-3", subjectKindDevice, "9f2c41d07a6be853", nil)
		require.NoError(t, err)
		assert.JSONEq(t, `{
			"version": 1,
			"timestamp": "2026-10-13T08:12:04Z",
			"resource": "pier-point/hvac/ctrl-3",
			"subject": {"kind": "device"},
			"hash": "9f2c41d07a6be853",
			"checks": []
		}`, string(b))
	})

	t.Run("checks use proto names and enum names", func(t *testing.T) {
		c := &healthpb.HealthCheck{
			Id:             "c1",
			DisplayName:    "Fan status",
			Normality:      healthpb.HealthCheck_ABNORMAL,
			OccupantImpact: healthpb.HealthCheck_LIFE,
		}
		b, err := buildHealthSet(now, "fcu-1", subjectKindService, "h", []*healthpb.HealthCheck{c})
		require.NoError(t, err)
		var got struct {
			Subject healthSubject    `json:"subject"`
			Checks  []map[string]any `json:"checks"`
		}
		require.NoError(t, json.Unmarshal(b, &got))
		assert.Equal(t, subjectKindService, got.Subject.Kind)
		require.Len(t, got.Checks, 1)
		assert.Equal(t, map[string]any{
			"id":              "c1",
			"display_name":    "Fan status",
			"normality":       "ABNORMAL",
			"occupant_impact": "LIFE",
		}, got.Checks[0])
	})
}

func TestBuildHealthManifest(t *testing.T) {
	now := time.Date(2026, 10, 13, 9, 15, 0, 0, time.UTC)
	b, err := buildHealthManifest(now, nil)
	require.NoError(t, err)
	assert.JSONEq(t, `{"version": 1, "timestamp": "2026-10-13T09:15:00Z", "resources": []}`, string(b))

	b, err = buildHealthManifest(now, []manifestEntry{{NameHash: "f5b04fb7c61623cf", Hash: "9f2c41d07a6be853"}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"version": 1, "timestamp": "2026-10-13T09:15:00Z", "resources": [
		{"nameHash": "f5b04fb7c61623cf", "hash": "9f2c41d07a6be853"}
	]}`, string(b))
}

func TestSubjectKind(t *testing.T) {
	tests := map[metadatapb.Metadata_DeviceType]string{
		metadatapb.Metadata_DEVICE_TYPE_UNSPECIFIED: subjectKindDevice,
		metadatapb.Metadata_NODE:                    subjectKindNode,
		metadatapb.Metadata_GATEWAY:                 subjectKindNode,
		metadatapb.Metadata_HUB:                     subjectKindNode,
		metadatapb.Metadata_SERVICE:                 subjectKindService,
	}
	for typ, want := range tests {
		assert.Equal(t, want, subjectKind(typ), typ.String())
	}
}

func TestDailySlot(t *testing.T) {
	slot := dailySlot("pier-point/hvac/ctrl-3")
	assert.Equal(t, slot, dailySlot("pier-point/hvac/ctrl-3"), "stable")
	assert.GreaterOrEqual(t, slot, time.Duration(0))
	assert.Less(t, slot, 24*time.Hour)
	assert.Zero(t, slot%time.Minute)

	// spread across the day
	seen := map[time.Duration]bool{}
	for i := range 100 {
		seen[dailySlot(string(rune('a'+i%26))+time.Duration(i).String())] = true
	}
	assert.Greater(t, len(seen), 90)
}

func TestHealthTopics(t *testing.T) {
	assert.Equal(t, "tlm/bos/health", healthTopic("tlm"))
	assert.Equal(t, "tlm/bos/health", healthTopic("tlm/"))
	assert.Equal(t, "tlm/bos/health-manifest", healthManifestTopic("tlm"))
}
