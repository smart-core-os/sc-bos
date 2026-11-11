package opcua

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
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

func (h *healthTestHarness) assertNormality(t *testing.T, checkId string, expected healthpb.HealthCheck_Normality) {
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
		normalValue   float64
		errorCode     string
		expectFault   bool
		expectNormal  healthpb.HealthCheck_Normality
		validateFault bool
	}{
		{
			name:         "value equals normal",
			value:        20.0,
			normalValue:  20.0,
			errorCode:    "TEMP_ABNORMAL",
			expectFault:  false,
			expectNormal: healthpb.HealthCheck_NORMAL,
		},
		{
			name:          "value above normal",
			value:         25.0,
			normalValue:   20.0,
			errorCode:     "TEMP_HIGH",
			expectFault:   true,
			expectNormal:  healthpb.HealthCheck_ABNORMAL,
			validateFault: true,
		},
		{
			name:         "value below normal",
			value:        15.0,
			normalValue:  20.0,
			errorCode:    "TEMP_LOW",
			expectFault:  true,
			expectNormal: healthpb.HealthCheck_ABNORMAL,
		},
		{
			name:         "value very close to normal (within tolerance)",
			value:        20.0000001,
			normalValue:  20.0,
			errorCode:    "TEMP_ABNORMAL",
			expectFault:  false,
			expectNormal: healthpb.HealthCheck_NORMAL,
		},
		{
			name:         "value different from normal",
			value:        30.0,
			normalValue:  20.0,
			errorCode:    "TEMP_ABNORMAL",
			expectFault:  true,
			expectNormal: healthpb.HealthCheck_ABNORMAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON := `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "temp-check",
					"displayName": "Temperature Sensor Fault",
					"description": "Detects temperature sensor faults",
					"summary": "Temperature sensor fault detected",
					"errorCode": "` + tt.errorCode + `",
					"nodeId": "ns=2;s=Temperature",
					"normalValue": ` + strconv.FormatFloat(tt.normalValue, 'f', -1, 64) + `
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
			"displayName": "Temperature Sensor Fault",
			"description": "Detects temperature sensor faults",
			"summary": "Temperature sensor error",
			"errorCode": "TEMP_ABNORMAL",
			"nodeId": "ns=2;s=Temperature",
			"normalValue": 20.0
		}]
	}`

	h := setupHealthTestHarness(t, configJSON)

	// Value different from normal - should raise fault
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(35.0))
	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_ABNORMAL)

	// Value equals normal - should clear fault
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))
	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_NORMAL)

	// Value different from normal again - should raise fault
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(5.0))
	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_ABNORMAL)

	// Value equals normal again - should clear fault
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))
	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_NORMAL)
}

func TestHealthCheck_MultipleChecks(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [
			{
				"id": "temp-check",
				"displayName": "Temperature Sensor Fault",
				"description": "Detects temperature sensor faults",
				"summary": "Temperature sensor fault detected",
				"errorCode": "TEMP_ABNORMAL",
				"nodeId": "ns=2;s=Temperature",
				"normalValue": 20.0
			},
			{
				"id": "pressure-check",
				"displayName": "Pressure Sensor Fault",
				"description": "Detects pressure sensor faults",
				"summary": "Pressure sensor fault detected",
				"errorCode": "PRESSURE_ABNORMAL",
				"nodeId": "ns=2;s=Pressure",
				"normalValue": 150.0
			}
		]
	}`

	h := setupHealthTestHarness(t, configJSON)

	// Temperature abnormal, pressure normal
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(35.0))
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Pressure"), float32(150.0))

	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "pressure-check", 0)
	h.assertNormality(t, "pressure-check", healthpb.HealthCheck_NORMAL)

	// Both abnormal
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Pressure"), float32(250.0))

	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "pressure-check", 1)
	h.assertNormality(t, "pressure-check", healthpb.HealthCheck_ABNORMAL)

	// Both normal
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Pressure"), float32(150.0))

	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_NORMAL)
	h.assertFaultCount(t, "pressure-check", 0)
	h.assertNormality(t, "pressure-check", healthpb.HealthCheck_NORMAL)
}

func TestHealthCheck_MultipleChecksOnSameNode(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [
			{
				"id": "temp-warning",
				"displayName": "Temperature Sensor Warning",
				"description": "Detects minor temperature sensor faults",
				"summary": "Temperature sensor warning",
				"errorCode": "TEMP_WARNING",
				"nodeId": "ns=2;s=Temperature",
				"normalValue": 20.0
			},
			{
				"id": "temp-critical",
				"displayName": "Temperature Sensor Critical Fault",
				"description": "Detects critical temperature sensor faults",
				"summary": "Temperature sensor critical fault",
				"errorCode": "TEMP_CRITICAL",
				"nodeId": "ns=2;s=Temperature",
				"normalValue": 25.0
			}
		]
	}`

	h := setupHealthTestHarness(t, configJSON)

	// Value = 20.0: matches temp-warning normal, doesn't match temp-critical normal
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0))

	h.assertFaultCount(t, "temp-warning", 0)
	h.assertNormality(t, "temp-warning", healthpb.HealthCheck_NORMAL)
	h.assertFaultCount(t, "temp-critical", 1)
	h.assertNormality(t, "temp-critical", healthpb.HealthCheck_ABNORMAL)

	// Value = 35.0: doesn't match either normal value
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(35.0))

	h.assertFaultCount(t, "temp-warning", 1)
	h.assertNormality(t, "temp-warning", healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "temp-critical", 1)
	h.assertNormality(t, "temp-critical", healthpb.HealthCheck_ABNORMAL)

	// Value = 25.0: doesn't match temp-warning normal, matches temp-critical normal
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(25.0))

	h.assertFaultCount(t, "temp-warning", 1)
	h.assertNormality(t, "temp-warning", healthpb.HealthCheck_ABNORMAL)
	h.assertFaultCount(t, "temp-critical", 0)
	h.assertNormality(t, "temp-critical", healthpb.HealthCheck_NORMAL)
}

func TestHealthCheck_ToleranceHandling(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [{
			"id": "temp-check",
			"displayName": "Temperature Sensor Fault",
			"description": "Detects temperature sensor faults with tolerance",
			"summary": "Temperature sensor fault detected",
			"errorCode": "TEMP_ABNORMAL",
			"nodeId": "ns=2;s=Temperature",
			"normalValue": 20.0
		}]
	}`

	h := setupHealthTestHarness(t, configJSON)

	// Value within float tolerance - should be treated as normal
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.0000001))
	h.assertFaultCount(t, "temp-check", 0)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_NORMAL)

	// Value outside tolerance - should raise fault
	h.health.handleEvent(t.Context(), mustParseNodeID("ns=2;s=Temperature"), float32(20.1))
	h.assertFaultCount(t, "temp-check", 1)
	h.assertNormality(t, "temp-check", healthpb.HealthCheck_ABNORMAL)
}

// currentFault returns the one fault on checkId, failing the test if there is not exactly one.
func (h *healthTestHarness) currentFault(t *testing.T, checkId string) *healthpb.HealthCheck_Error {
	t.Helper()
	for _, check := range h.getHealthChecks(t) {
		if check.Id == "example:"+checkId {
			faults := check.GetFaults().GetCurrentFaults()
			require.Len(t, faults, 1)
			return faults[0]
		}
	}
	t.Fatalf("check %s not found", checkId)
	return nil
}

// TestHealthCheck_FaultUpdate checks a move from one abnormal value to another changes the
// fault, so it is published, where it used to repeat the same text and look unchanged: a board
// going from Warning to Alarm read the same in Connect as one sitting at Warning.
func TestHealthCheck_FaultUpdate(t *testing.T) {
	configJSON := `{
		"kind": "smartcore.trait.Health",
		"checks": [{
			"id": "temp-check",
			"displayName": "Temperature Sensor Fault",
			"description": "Detects temperature sensor faults",
			"summary": "Temperature sensor fault detected",
			"errorCode": "TEMP_ABNORMAL",
			"nodeId": "ns=2;s=Temperature",
			"normalValue": 20.0
		}]
	}`

	h := setupHealthTestHarness(t, configJSON)
	node := mustParseNodeID("ns=2;s=Temperature")

	h.health.handleEvent(t.Context(), node, float32(35.0))
	first := h.currentFault(t, "temp-check")
	require.Contains(t, first.DetailsText, "read 35,")

	// the same value again leaves the fault exactly as it was, so nothing is republished
	h.health.handleEvent(t.Context(), node, float32(35.0))
	if diff := cmp.Diff(first, h.currentFault(t, "temp-check"), protocmp.Transform()); diff != "" {
		t.Errorf("a repeated value should leave the fault unchanged (-want +got):\n%s", diff)
	}

	// another abnormal value replaces the fault rather than adding a second one
	h.health.handleEvent(t.Context(), node, float32(40.0))
	second := h.currentFault(t, "temp-check")
	require.Contains(t, second.DetailsText, "read 40,")
	require.NotEqual(t, first.DetailsText, second.DetailsText)
	if diff := cmp.Diff(first.Code, second.Code, protocmp.Transform()); diff != "" {
		t.Errorf("fault code should remain the same (-want +got):\n%s", diff)
	}
	require.Equal(t, first.SummaryText, second.SummaryText, "the configured summary is kept as is")
}

// TestHealthCheck_FaultDetails checks a fault says what the point read, what was expected,
// and which point it was. An IEC 61850 Health point reads 2 for Warning and 3 for Alarm, so
// the value is what tells an operator how urgent the fault is.
func TestHealthCheck_FaultDetails(t *testing.T) {
	const nodeId = `ns=2;s=LDB_X.LPHD1.EEHealth`
	tests := []struct {
		name        string
		pointName   string
		enum        string
		value       any
		wantDetails string
	}{
		{
			name:        "no enum",
			pointName:   "LPHD1.EEHealth",
			value:       int16(2),
			wantDetails: `Point LPHD1.EEHealth read 2, expected 1. NodeID: ns=2;s=LDB_X.LPHD1.EEHealth`,
		},
		{
			name:        "enum labels both values",
			pointName:   "LPHD1.EEHealth",
			enum:        `{"1": "Ok", "2": "Warning", "3": "Alarm"}`,
			value:       int16(3),
			wantDetails: `Point LPHD1.EEHealth read 3 (Alarm), expected 1 (Ok). NodeID: ns=2;s=LDB_X.LPHD1.EEHealth`,
		},
		{
			name:        "value with no enum entry",
			pointName:   "LPHD1.EEHealth",
			enum:        `{"1": "Ok", "2": "Warning", "3": "Alarm"}`,
			value:       int16(7),
			wantDetails: `Point LPHD1.EEHealth read 7, expected 1 (Ok). NodeID: ns=2;s=LDB_X.LPHD1.EEHealth`,
		},
		{
			// 2.5 must not be truncated into the label for 2
			name:        "fractional value",
			pointName:   "LPHD1.EEHealth",
			enum:        `{"1": "Ok", "2": "Warning", "3": "Alarm"}`,
			value:       float32(2.5),
			wantDetails: `Point LPHD1.EEHealth read 2.5, expected 1 (Ok). NodeID: ns=2;s=LDB_X.LPHD1.EEHealth`,
		},
		{
			// rendered at the precision the point reported, not as 20.100000381469727
			name:        "float point",
			pointName:   "Temp",
			value:       float32(20.1),
			wantDetails: `Point Temp read 20.1, expected 1. NodeID: ns=2;s=LDB_X.LPHD1.EEHealth`,
		},
		{
			name:        "no name leads with the node id",
			value:       int16(2),
			wantDetails: `Point ns=2;s=LDB_X.LPHD1.EEHealth read 2, expected 1.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extra := ""
			if tt.pointName != "" {
				extra += `"name": "` + tt.pointName + `",`
			}
			if tt.enum != "" {
				extra += `"enum": ` + tt.enum + `,`
			}
			configJSON := `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "board-health",
					"displayName": "Board Health",
					"description": "Checks the board reports itself healthy",
					"summary": "Board health is not OK",
					"errorCode": "BoardHealth",
					"nodeId": "` + nodeId + `",` + extra + `
					"normalValue": 1
				}]
			}`

			h := setupHealthTestHarness(t, configJSON)
			h.health.handleEvent(t.Context(), mustParseNodeID(nodeId), tt.value)

			fault := h.currentFault(t, "board-health")
			require.Equal(t, tt.wantDetails, fault.DetailsText)
			require.Equal(t, "Board health is not OK", fault.SummaryText)
			require.Equal(t, "BoardHealth", fault.GetCode().GetCode())
			require.Equal(t, SystemName, fault.GetCode().GetSystem())
		})
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
					"displayName": "Device Fault Detection",
					"description": "Detects device faults and errors",
					"summary": "Device fault detected",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test",
					"normalValue": 100.0
				}]
			}`,
			expectError: false,
		},
		{
			name: "missing id",
			configJSON: `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"displayName": "Device Fault Detection",
					"description": "Detects device faults and errors",
					"summary": "Device fault detected",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test",
					"normalValue": 100.0
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
					"description": "Detects device faults and errors",
					"summary": "Device fault detected",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test",
					"normalValue": 100.0
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
					"displayName": "Device Fault Detection",
					"description": "Detects device faults and errors",
					"summary": "Device fault detected",
					"errorCode": "TEST_ERROR",
					"normalValue": 100.0
				}]
			}`,
			expectError: true,
			errorMsg:    "nodeId is required",
		},
		{
			name: "missing normalValue",
			configJSON: `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "test",
					"displayName": "Device Fault Detection",
					"description": "Detects device faults and errors",
					"summary": "Device fault detected",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test"
				}]
			}`,
			expectError: true,
			errorMsg:    "normalValue is required",
		},
		{
			name: "missing summary",
			configJSON: `{
				"kind": "smartcore.trait.Health",
				"checks": [{
					"id": "test",
					"displayName": "Device Fault Detection",
					"description": "Detects device faults and errors",
					"errorCode": "TEST_ERROR",
					"nodeId": "ns=2;s=Test",
					"normalValue": 100.0
				}]
			}`,
			expectError: true,
			errorMsg:    "summary is required",
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

// TestSetPointRead checks how a point's status code reaches the reliability report.
// The health code is hex so it can be matched against the code the driver logs; it used to
// be decimal, rendering 0x480 as "1152" and a Bad code as "2150629376".
func TestSetPointRead(t *testing.T) {
	tests := []struct {
		name        string
		set         func(h *testHarness, nodeId string, status ua.StatusCode)
		status      ua.StatusCode
		wantState   healthpb.HealthCheck_Reliability_State
		wantCode    string
		wantSummary string
	}{
		{
			name: "bad",
			set: func(h *testHarness, nodeId string, status ua.StatusCode) {
				setPointReadNotOk(h.ctx, nodeId, status, h.fc)
			},
			status:      ua.StatusBadNodeIDUnknown,
			wantState:   healthpb.HealthCheck_Reliability_BAD_RESPONSE,
			wantCode:    "0x80340000",
			wantSummary: "non OK status",
		},
		{
			name: "uncertain",
			set: func(h *testHarness, nodeId string, status ua.StatusCode) {
				setPointReadUncertain(h.ctx, nodeId, status, h.fc)
			},
			status:      ua.StatusUncertainSimulatedValue,
			wantState:   healthpb.HealthCheck_Reliability_UNRELIABLE,
			wantCode:    "0x42090000",
			wantSummary: "uncertain status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupTestHarness(t)
			const nodeId = "ns=2;s=Tag1"

			tt.set(h, nodeId, tt.status)

			rel := h.getHealthChecks(t)[0].GetReliability()
			require.NotNil(t, rel)
			require.Equal(t, tt.wantState, rel.State)
			require.NotNil(t, rel.LastError)
			require.Equal(t, tt.wantCode, rel.LastError.Code.Code)
			require.Equal(t, SystemName, rel.LastError.Code.System)
			require.Contains(t, rel.LastError.SummaryText, tt.wantSummary)
			require.Contains(t, rel.LastError.DetailsText, nodeId)
		})
	}
}

// Test_pointHealth checks how the subscribe state of a device's points reaches its fault check.
//
// The aggregation is what needs pinning down. Faults are keyed on system and code, so a fault
// raised per point would have every point overwriting its neighbours; one fault per state,
// naming the points it covers, is the only shape that survives. And each has to be removed
// once its bucket empties, which is the first thing in this driver that ever clears a fault.
func Test_pointHealth(t *testing.T) {
	const nodeA, nodeB = "ns=2;s=TagA", "ns=2;s=TagB"
	h := setupTestHarness(t)
	p := newPointHealth(h.fc)

	// the harness mirrors a check onto the device when it is updated rather than when it is
	// created, so before the first write there is no check to read. That is a quirk of the
	// harness, hence tolerating an empty list rather than requiring one check.
	faults := func() map[string]*healthpb.HealthCheck_Error {
		t.Helper()
		out := make(map[string]*healthpb.HealthCheck_Error)
		for _, c := range h.getHealthChecks(t) {
			for _, f := range c.GetFaults().GetCurrentFaults() {
				out[f.GetCode().GetCode()] = f
			}
		}
		return out
	}
	normality := func() healthpb.HealthCheck_Normality {
		t.Helper()
		checks := h.getHealthChecks(t)
		require.Len(t, checks, 1)
		return checks[0].GetNormality()
	}

	require.Empty(t, faults(), "nothing has been said about any point yet")

	// one point failing for a reason that may go away
	p.setFailing(nodeA, ua.StatusBadTimeout)
	got := faults()
	require.Len(t, got, 1, "a point being retried is not a config fault as well")
	require.Contains(t, got, PointSubscribeError)
	require.Contains(t, got[PointSubscribeError].SummaryText, "1 of")
	require.Contains(t, got[PointSubscribeError].DetailsText, nodeA)
	require.Contains(t, got[PointSubscribeError].DetailsText, ua.StatusBadTimeout.Error())
	require.Equal(t, SystemName, got[PointSubscribeError].GetCode().GetSystem())
	require.Equal(t, healthpb.HealthCheck_ABNORMAL, normality())

	// a second one joins it rather than replacing it
	p.setFailing(nodeB, ua.StatusBadNoCommunication)
	got = faults()
	require.Len(t, got, 1)
	details := got[PointSubscribeError].DetailsText
	require.Contains(t, got[PointSubscribeError].SummaryText, "2 of")
	require.Contains(t, details, nodeA)
	require.Contains(t, details, nodeB)
	// sorted, so an unchanged set of failures renders identically and costs nothing to rewrite
	require.Less(t, strings.Index(details, nodeA), strings.Index(details, nodeB))

	// giving up on one moves it from being retried to being a config fault, and leaves the
	// other where it was
	p.setPermanent(nodeB, ua.StatusBadNodeIDUnknown)
	got = faults()
	require.Len(t, got, 2)
	require.Contains(t, got[PointSubscribeError].DetailsText, nodeA)
	require.NotContains(t, got[PointSubscribeError].DetailsText, nodeB)
	require.Contains(t, got[DeviceConfigError].DetailsText, nodeB)
	require.Contains(t, got[DeviceConfigError].DetailsText, ua.StatusBadNodeIDUnknown.Error())

	// the retried point comes good: its fault is removed, the config fault is untouched
	p.setOk(nodeA)
	got = faults()
	require.Len(t, got, 1)
	require.NotContains(t, got, PointSubscribeError)
	require.Contains(t, got[DeviceConfigError].DetailsText, nodeB)
	require.Equal(t, healthpb.HealthCheck_ABNORMAL, normality())

	// and a point we had given up on can still recover, if a config reload brings it back
	p.setOk(nodeB)
	require.Empty(t, faults())
	require.Equal(t, healthpb.HealthCheck_NORMAL, normality())
}

// Test_renderPoints checks identical state renders identically, which is what makes rewriting
// the faults on every change cheap: checkBase.write drops an update that changes nothing.
func Test_renderPoints(t *testing.T) {
	first := renderPoints(map[string]string{"ns=2;s=B": "bad b", "ns=2;s=A": "bad a"})
	second := renderPoints(map[string]string{"ns=2;s=A": "bad a", "ns=2;s=B": "bad b"})
	require.Equal(t, first, second, "the same failures inserted in a different order should render the same")
	require.Equal(t, "ns=2;s=A: bad a\nns=2;s=B: bad b", first)
	require.Empty(t, renderPoints(nil))
}

// countingFaultCheck reports the faults on a check and how many updates the registry has
// published for it, which is the cost applyBatch exists to avoid.
func countingFaultCheck(t *testing.T) (*healthpb.FaultCheck, func() (map[string]string, int)) {
	t.Helper()
	var latest *healthpb.HealthCheck
	updates := 0
	reg := healthpb.NewRegistry(
		healthpb.WithOnCheckUpdate(func(_ string, c *healthpb.HealthCheck) {
			updates++
			latest = c
		}),
	)
	check := getDeviceHealthCheck(healthpb.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED, healthpb.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED)
	fc, err := reg.ForOwner("test").NewFaultCheck("test-device", check)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)

	// read is called from the test goroutine only, as are the writes above, so there is
	// nothing here to synchronise
	read := func() (map[string]string, int) {
		faults := make(map[string]string)
		for _, f := range latest.GetFaults().GetCurrentFaults() {
			faults[f.GetCode().GetCode()] = f.GetSummaryText() + "\n" + f.GetDetailsText()
		}
		return faults, updates
	}
	return fc, read
}

// Test_pointHealth_applyBatch checks a device's whole subscribe outcome commits as one update,
// reading exactly as the setters it replaces.
//
// commit rewrites both faults and pushes the check to the registry every time it runs. With a
// subscription per point those calls were spread over as many goroutines; with one per device
// they arrive together, so a 200-point device would publish 200 growing versions of the same
// fault per attempt, every one of them visible over the devices API. What must not change is
// what an operator ends up reading, hence comparing the text rather than just the count.
func Test_pointHealth_applyBatch(t *testing.T) {
	const nodeA, nodeB, nodeC, nodeD, nodeE = "ns=2;s=A", "ns=2;s=B", "ns=2;s=C", "ns=2;s=D", "ns=2;s=E"
	timeout, unknown := ua.StatusBadTimeout, ua.StatusBadNodeIDUnknown

	sequential, readSequential := countingFaultCheck(t)
	one := newPointHealth(sequential)
	one.setOk(nodeA)
	one.setFailing(nodeB, timeout)
	one.setFailing(nodeC, timeout)
	one.setPermanent(nodeD, unknown)
	one.setPermanent(nodeE, unknown)

	batched, readBatched := countingFaultCheck(t)
	all := newPointHealth(batched)
	all.applyBatch([]string{nodeA},
		map[string]error{nodeB: timeout, nodeC: timeout},
		map[string]error{nodeD: unknown, nodeE: unknown})

	sequentialFaults, sequentialUpdates := readSequential()
	batchedFaults, batchedUpdates := readBatched()

	require.Equal(t, sequentialFaults, batchedFaults, "the batch must read exactly as the setters it replaces")
	require.Len(t, batchedFaults, 2, "one fault per state, each naming the points it covers")
	// two rather than one because commit writes each fault code separately, which is
	// healthpb's API rather than anything we choose. The point is that it is per fault code
	// and not per point: five points still cost the same two writes fifty would.
	require.Equal(t, 2, batchedUpdates, "a device's whole outcome commits once per fault code")
	require.Greater(t, sequentialUpdates, batchedUpdates,
		"an update per point is the cost this exists to avoid")
}
