package opcua

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

type healthTestHarness struct {
	*testHarness
	health *Health
}

func setupHealthTestHarness(t *testing.T, configJSON string) *healthTestHarness {
	h := setupTestHarness(t)

	logger := zaptest.NewLogger(t)
	rawTrait := config.RawTrait{
		Trait: config.Trait{
			Name: "smartcore.trait.Health",
		},
		Raw: json.RawMessage(configJSON),
	}

	health, err := newHealth(rawTrait, logger)
	require.NoError(t, err)
	require.NotEmpty(t, health.cfg.Checks, "health config should have at least one check")

	reg := newTestRegistry(h.devs)
	healthChecks := reg.ForOwner("example")

	for i := range health.cfg.Checks {
		check := &health.cfg.Checks[i]
		hc := getDeviceErrorCheck(*check)
		fc, err := healthChecks.NewFaultCheck("opcua-device-1", hc)
		require.NoError(t, err)
		health.errorChecks[check.Id] = fc
		t.Cleanup(fc.Dispose)
	}

	require.Len(t, health.errorChecks, len(health.cfg.Checks), "all health checks should have fault checks created")

	return &healthTestHarness{
		testHarness: h,
		health:      health,
	}
}

func (h *healthTestHarness) assertFaultCount(t *testing.T, checkId string, expectedCount int) {
	checks := h.getHealthChecks(t)
	fullId := "example:" + checkId

	var allIds []string
	for _, check := range checks {
		allIds = append(allIds, check.Id)
	}

	for _, check := range checks {
		if check.Id == fullId {
			faults := check.GetFaults().GetCurrentFaults()
			if diff := cmp.Diff(expectedCount, len(faults)); diff != "" {
				t.Errorf("fault count mismatch for check %s (-want +got):\n%s", checkId, diff)
			}
			return
		}
	}
	t.Errorf("check with id %s not found (looking for %s). Available checks: %v", checkId, fullId, allIds)
}

func (h *healthTestHarness) assertNormality(t *testing.T, checkId string, expected gen_healthpb.HealthCheck_Normality) {
	checks := h.getHealthChecks(t)
	fullId := "example:" + checkId
	for _, check := range checks {
		if check.Id == fullId {
			if diff := cmp.Diff(expected, check.Normality, protocmp.Transform()); diff != "" {
				t.Errorf("normality mismatch for check %s (-want +got):\n%s", checkId, diff)
			}
			return
		}
	}
	t.Errorf("check with id %s not found (looking for %s)", checkId, fullId)
}

func TestHealthCheck_SingleValue(t *testing.T) {
	tests := []struct {
		name          string
		value         float32
		errorCode     string
		expectFault   bool
		expectNormal  gen_healthpb.HealthCheck_Normality
		validateFault bool
	}{
		{
			name:         "value within bounds",
			value:        20.0,
			errorCode:    "TEMP_OUT_OF_RANGE",
			expectFault:  false,
			expectNormal: gen_healthpb.HealthCheck_NORMAL,
		},
		{
			name:          "value above upper bound",
			value:         35.0,
			errorCode:     "TEMP_HIGH",
			expectFault:   true,
			expectNormal:  gen_healthpb.HealthCheck_ABNORMAL,
			validateFault: true,
		},
		{
			name:         "value below lower bound",
			value:        5.0,
			errorCode:    "TEMP_LOW",
			expectFault:  true,
			expectNormal: gen_healthpb.HealthCheck_ABNORMAL,
		},
		{
			name:         "value at lower bound",
			value:        10.0,
			errorCode:    "TEMP_OUT_OF_RANGE",
			expectFault:  false,
			expectNormal: gen_healthpb.HealthCheck_NORMAL,
		},
		{
			name:         "value at upper bound",
			value:        30.0,
			errorCode:    "TEMP_OUT_OF_RANGE",
			expectFault:  false,
			expectNormal: gen_healthpb.HealthCheck_NORMAL,
		},
		{
			name:         "just below lower bound",
			value:        9.999,
			errorCode:    "TEMP_OUT_OF_RANGE",
			expectFault:  true,
			expectNormal: gen_healthpb.HealthCheck_ABNORMAL,
		},
		{
			name:         "just above upper bound",
			value:        30.001,
			errorCode:    "TEMP_OUT_OF_RANGE",
			expectFault:  true,
			expectNormal: gen_healthpb.HealthCheck_ABNORMAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON := `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "temp-check",
					"displayName": "Temperature Check",
					"description": "Monitor temperature",
					"errorCode": "` + tt.errorCode + `",
					"nodeId": "ns=2;s=Temperature",
					"okLowerBound": 10.0,
					"okUpperBound": 30.0
				}]
			}`

			h := setupHealthTestHarness(t, configJSON)
			h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), tt.value)

			expectedCount := 0
			if tt.expectFault {
				expectedCount = 1
			}
			h.assertFaultCount(t, "temp-check", expectedCount)
			h.assertNormality(t, "temp-check", tt.expectNormal)

			if tt.validateFault && tt.expectFault {
				checks := h.getHealthChecks(t)
				for _, check := range checks {
					if check.Id == "example:temp-check" {
						faults := check.GetFaults().GetCurrentFaults()
						require.Len(t, faults, 1)
						require.Equal(t, tt.errorCode, faults[0].Code.Code)
						require.Equal(t, SystemName, faults[0].Code.System)
					}
				}
			}
		})
	}
}

func TestHealthCheck_FaultLifecycle(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [{
			"id": "temp-check",
			"displayName": "Temperature Check",
			"description": "Monitor temperature",
			"errorCode": "TEMP_OUT_OF_RANGE",
			"nodeId": "ns=2;s=Temperature",
			"okLowerBound": 10.0,
			"okUpperBound": 30.0
		}]
	}`

	h := setupHealthTestHarness(t, configJSON)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(35.0))
	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_ABNORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))
	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_NORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(5.0))
	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_ABNORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))
	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_NORMAL)
}

func TestHealthCheck_MultipleChecks(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [
			{
				"id": "temp-check",
				"displayName": "Temperature Check",
				"description": "Monitor temperature",
				"errorCode": "TEMP_OUT_OF_RANGE",
				"nodeId": "ns=2;s=Temperature",
				"okLowerBound": 10.0,
				"okUpperBound": 30.0
			},
			{
				"id": "pressure-check",
				"displayName": "Pressure Check",
				"description": "Monitor pressure",
				"errorCode": "PRESSURE_OUT_OF_RANGE",
				"nodeId": "ns=2;s=Pressure",
				"okLowerBound": 100.0,
				"okUpperBound": 200.0
			}
		]
	}`

	h := setupHealthTestHarness(t, configJSON)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(35.0))
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Pressure"), float32(150.0))

	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "pressure-check", 0)
	h.assertNormality(t, "pressure-check", gen_healthpb.HealthCheck_NORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Pressure"), float32(250.0))

	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "pressure-check", 1)
	h.assertNormality(t, "pressure-check", gen_healthpb.HealthCheck_ABNORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Pressure"), float32(150.0))

	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_NORMAL)
	h.assertFaultCount(t, "pressure-check", 0)
	h.assertNormality(t, "pressure-check", gen_healthpb.HealthCheck_NORMAL)
}

func TestHealthCheck_MultipleChecksOnSameNode(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [
			{
				"id": "temp-warning",
				"displayName": "Temperature Warning",
				"description": "Temperature warning threshold",
				"errorCode": "TEMP_WARNING",
				"nodeId": "ns=2;s=Temperature",
				"okLowerBound": 15.0,
				"okUpperBound": 25.0
			},
			{
				"id": "temp-critical",
				"displayName": "Temperature Critical",
				"description": "Temperature critical threshold",
				"errorCode": "TEMP_CRITICAL",
				"nodeId": "ns=2;s=Temperature",
				"okLowerBound": 10.0,
				"okUpperBound": 30.0
			}
		]
	}`

	h := setupHealthTestHarness(t, configJSON)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(27.0))

	h.assertFaultCount(t, "temp-warning", 1)
	h.assertNormality(t, "temp-warning", gen_healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "temp-critical", 0)
	h.assertNormality(t, "temp-critical", gen_healthpb.HealthCheck_NORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(35.0))

	h.assertFaultCount(t, "temp-warning", 1)
	h.assertNormality(t, "temp-warning", gen_healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "temp-critical", 1)
	h.assertNormality(t, "temp-critical", gen_healthpb.HealthCheck_ABNORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))

	h.assertFaultCount(t, "temp-warning", 0)
	h.assertNormality(t, "temp-warning", gen_healthpb.HealthCheck_NORMAL)
	h.assertFaultCount(t, "temp-critical", 0)
	h.assertNormality(t, "temp-critical", gen_healthpb.HealthCheck_NORMAL)
}

func TestHealthCheck_InfinityBounds(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [{
			"id": "temp-check",
			"displayName": "Temperature Check",
			"description": "Monitor temperature",
			"errorCode": "TEMP_HIGH",
			"nodeId": "ns=2;s=Temperature",
			"okUpperBound": 30.0
		}]
	}`

	h := setupHealthTestHarness(t, configJSON)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(-1000.0))
	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_NORMAL)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(100.0))
	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", gen_healthpb.HealthCheck_ABNORMAL)
}

func TestHealthCheck_FaultUpdate(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [{
			"id": "temp-check",
			"displayName": "Temperature Check",
			"description": "Monitor temperature",
			"errorCode": "TEMP_OUT_OF_RANGE",
			"nodeId": "ns=2;s=Temperature",
			"okLowerBound": 10.0,
			"okUpperBound": 30.0
		}]
	}`

	h := setupHealthTestHarness(t, configJSON)

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(35.0))
	h.assertFaultCount(t, "temp-check", 1)

	checks := h.getHealthChecks(t)
	var firstFaultCode string
	for _, check := range checks {
		if check.Id == "example:temp-check" {
			faults := check.GetFaults().GetCurrentFaults()
			require.Len(t, faults, 1)
			firstFaultCode = faults[0].Code.Code
		}
	}

	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(40.0))
	h.assertFaultCount(t, "temp-check", 1)

	checks = h.getHealthChecks(t)
	for _, check := range checks {
		if check.Id == "example:temp-check" {
			faults := check.GetFaults().GetCurrentFaults()
			require.Len(t, faults, 1)

			if diff := cmp.Diff(firstFaultCode, faults[0].Code.Code); diff != "" {
				t.Errorf("fault code should remain the same (-want +got):\n%s", diff)
			}
		}
	}
}

func TestNewHealth_ValidationCalled(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			configJSON: `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "test",
					"displayName": "Test",
					"description": "Test check",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test"
				}]
			}`,
			expectError: false,
		},
		{
			name: "missing id",
			configJSON: `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"displayName": "Test",
					"description": "Test check",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test"
				}]
			}`,
			expectError: true,
			errorMsg:    "id is required",
		},
		{
			name: "missing displayName",
			configJSON: `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "test",
					"description": "Test check",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test"
				}]
			}`,
			expectError: true,
			errorMsg:    "displayName is required",
		},
		{
			name: "missing nodeId",
			configJSON: `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "test",
					"displayName": "Test",
					"description": "Test check",
					"errorCode": "TEST_ERROR"
				}]
			}`,
			expectError: true,
			errorMsg:    "nodeId is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			rawTrait := config.RawTrait{
				Trait: config.Trait{
					Name: "smartcore.trait.Health",
				},
				Raw: json.RawMessage(tt.configJSON),
			}

			_, err := newHealth(rawTrait, logger)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
