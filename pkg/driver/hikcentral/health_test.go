package hikcentral

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/api"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func newTestRegistry(devs *devicespb.Collection) *healthpb.Registry {

	return healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(name string, c *gen.HealthCheck) *gen.HealthCheck {
			_, _ = devs.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}), resource.WithCreateIfAbsent(), resource.WithExpectAbsent())
			return nil
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *gen.HealthCheck) {
			_, _ = devs.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}))
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			_, _ = devs.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
			}), resource.WithAllowMissing(true))
		}),
	)
}

type testHarness struct {
	devs   *devicespb.Collection
	client gen.DevicesApiClient
	fc     *healthpb.FaultCheck
	ctx    context.Context
}

func setupTestHarness(t *testing.T) *testHarness {
	devs := devicespb.NewCollection()
	server := devices.NewServer(devicesServerModel{Collection: devs})
	deviceName := "hikcentral-camera-1"
	reg := newTestRegistry(devs)
	healthChecks := reg.ForOwner("example")

	_, _ = devs.Update(&gen.Device{Name: deviceName}, resource.WithCreateIfAbsent())

	fc, err := healthChecks.NewFaultCheck(deviceName, deviceHealthCheck)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)

	return &testHarness{
		devs:   devs,
		client: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, server)),
		fc:     fc,
		ctx:    context.Background(),
	}
}

func (h *testHarness) updateFaults(faults allFaults) {
	updateDeviceFaults(faults, h.fc)
}

func (h *testHarness) updateReliability(err error) {
	updateReliability(h.ctx, err, h.fc)
}

func (h *testHarness) getHealthChecks(t *testing.T) []*gen.HealthCheck {
	deviceList, err := h.client.ListDevices(context.TODO(), &gen.ListDevicesRequest{})
	require.NoError(t, err)
	require.Len(t, deviceList.Devices, 1)
	return deviceList.Devices[0].GetHealthChecks()
}

func (h *testHarness) assertFaults(t *testing.T, expectedCount int, normality gen.HealthCheck_Normality) {
	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, normality, checks[0].Normality)
	require.Len(t, checks[0].GetFaults().CurrentFaults, expectedCount)
}

func TestHikcentralFaults(t *testing.T) {
	tests := []struct {
		name          string
		faults        allFaults
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "no errors",
			faults:        allFaults{},
			expectedCount: 0,
			expectedCodes: nil,
		},
		{
			name: "single fault - video loss",
			faults: allFaults{
				api.VideoLossAlarm: true,
			},
			expectedCount: 1,
			expectedCodes: []string{api.VideoLossAlarm},
		},
		{
			name: "single fault - video tampering",
			faults: allFaults{
				api.VideoTamperingAlarm: true,
			},
			expectedCount: 1,
			expectedCodes: []string{api.VideoTamperingAlarm},
		},
		{
			name: "double fault - video loss and tampering",
			faults: allFaults{
				api.VideoLossAlarm:      true,
				api.VideoTamperingAlarm: true,
			},
			expectedCount: 2,
			expectedCodes: []string{api.VideoLossAlarm, api.VideoTamperingAlarm},
		},
		{
			name: "recording exception fault",
			faults: allFaults{
				api.CameraRecordingExceptionAlarm: true,
			},
			expectedCount: 1,
			expectedCodes: []string{api.CameraRecordingExceptionAlarm},
		},
		{
			name: "recording recovered is not a fault",
			faults: allFaults{
				api.CameraRecordingRecovered: true,
			},
			expectedCount: 0, // CameraRecordingRecovered should never appear as a fault
			expectedCodes: nil,
		},
		{
			name: "all faults (including recovered marker)",
			faults: allFaults{
				api.VideoLossAlarm:                true,
				api.VideoTamperingAlarm:           true,
				api.CameraRecordingExceptionAlarm: true,
				api.CameraRecordingRecovered:      true, // Not a fault, just a marker
			},
			expectedCount: 3, // Only 3 actual faults
			expectedCodes: []string{
				api.VideoLossAlarm,
				api.VideoTamperingAlarm,
				api.CameraRecordingExceptionAlarm,
				// CameraRecordingRecovered is NOT included as it's not a fault
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupTestHarness(t)
			h.updateFaults(tt.faults)

			checks := h.getHealthChecks(t)
			require.Len(t, checks, 1)

			faults := checks[0].GetFaults().GetCurrentFaults()
			if diff := cmp.Diff(tt.expectedCount, len(faults)); diff != "" {
				t.Errorf("unexpected fault count (-want +got):\n%s", diff)
			}

			// Collect all fault codes from the actual faults
			actualCodes := make([]string, len(faults))
			for i, fault := range faults {
				actualCodes[i] = fault.Code.Code
			}

			// Check that all expected codes are present (order may vary)
			for _, expectedCode := range tt.expectedCodes {
				found := false
				for _, actualCode := range actualCodes {
					if actualCode == expectedCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected fault code %s not found in actual codes: %v", expectedCode, actualCodes)
				}
			}
		})
	}
}

func TestFaultLifecycle(t *testing.T) {
	tests := []struct {
		name  string
		steps []struct {
			faults       allFaults
			faultCount   int
			normality    gen.HealthCheck_Normality
			expectedCode string // only for single fault cases
		}
	}{
		{
			name: "add multiple faults then clear all",
			steps: []struct {
				faults       allFaults
				faultCount   int
				normality    gen.HealthCheck_Normality
				expectedCode string
			}{
				{
					faults: allFaults{
						api.VideoLossAlarm:      true,
						api.VideoTamperingAlarm: true,
					},
					faultCount: 2,
					normality:  gen.HealthCheck_ABNORMAL,
				},
				{
					faults:     allFaults{},
					faultCount: 0,
					normality:  gen.HealthCheck_NORMAL,
				},
			},
		},
		{
			name: "add multiple faults then partial clear",
			steps: []struct {
				faults       allFaults
				faultCount   int
				normality    gen.HealthCheck_Normality
				expectedCode string
			}{
				{
					faults: allFaults{
						api.VideoLossAlarm:                true,
						api.VideoTamperingAlarm:           true,
						api.CameraRecordingExceptionAlarm: true,
					},
					faultCount: 3,
					normality:  gen.HealthCheck_ABNORMAL,
				},
				{
					faults: allFaults{
						api.VideoLossAlarm: true, // Keep only VideoLossAlarm
					},
					faultCount:   1,
					normality:    gen.HealthCheck_ABNORMAL,
					expectedCode: api.VideoLossAlarm,
				},
				{
					faults:     allFaults{},
					faultCount: 0,
					normality:  gen.HealthCheck_NORMAL,
				},
			},
		},
		{
			name: "transition between different single faults",
			steps: []struct {
				faults       allFaults
				faultCount   int
				normality    gen.HealthCheck_Normality
				expectedCode string
			}{
				{
					faults: allFaults{
						api.VideoLossAlarm: true,
					},
					faultCount:   1,
					normality:    gen.HealthCheck_ABNORMAL,
					expectedCode: api.VideoLossAlarm,
				},
				{
					faults: allFaults{
						api.VideoTamperingAlarm: true,
					},
					faultCount:   1,
					normality:    gen.HealthCheck_ABNORMAL,
					expectedCode: api.VideoTamperingAlarm,
				},
				{
					faults:     allFaults{},
					faultCount: 0,
					normality:  gen.HealthCheck_NORMAL,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupTestHarness(t)

			for i, step := range tt.steps {
				h.updateFaults(step.faults)

				checks := h.getHealthChecks(t)
				require.Len(t, checks, 1)

				if diff := cmp.Diff(step.normality, checks[0].Normality, protocmp.Transform()); diff != "" {
					t.Errorf("step %d: normality mismatch (-want +got):\n%s", i, diff)
				}

				faults := checks[0].GetFaults().GetCurrentFaults()
				if diff := cmp.Diff(step.faultCount, len(faults)); diff != "" {
					t.Errorf("step %d: fault count mismatch (-want +got):\n%s", i, diff)
				}

				if step.expectedCode != "" && len(faults) > 0 {
					if diff := cmp.Diff(step.expectedCode, faults[0].Code.Code); diff != "" {
						t.Errorf("step %d: fault code mismatch (-want +got):\n%s", i, diff)
					}
				}
			}
		})
	}
}

func TestFaultRemoval(t *testing.T) {
	h := setupTestHarness(t)

	// Step 1: Add multiple faults
	h.updateFaults(allFaults{
		api.VideoLossAlarm:                true,
		api.VideoTamperingAlarm:           true,
		api.CameraRecordingExceptionAlarm: true,
	})

	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults := checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(3, len(faults)); diff != "" {
		t.Errorf("initial fault count mismatch (-want +got):\n%s", diff)
	}

	// Step 2: Remove one fault (keep VideoLossAlarm and CameraRecordingExceptionAlarm)
	h.updateFaults(allFaults{
		api.VideoLossAlarm:                true,
		api.CameraRecordingExceptionAlarm: true,
	})

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(2, len(faults)); diff != "" {
		t.Errorf("fault count after removing one fault (-want +got):\n%s", diff)
	}

	// Verify VideoTamperingAlarm is gone
	for _, fault := range faults {
		if fault.Code.Code == api.VideoTamperingAlarm {
			t.Errorf("VideoTamperingAlarm should have been removed but is still present")
		}
	}

	// Step 3: Clear all faults
	h.updateFaults(allFaults{})

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(0, len(faults)); diff != "" {
		t.Errorf("fault count after clearing all faults mismatch (-want +got):\n%s", diff)
	}
}

func TestDriverRebootClearsFaults(t *testing.T) {
	h := setupTestHarness(t)

	// Add all faults (including recovered marker which is not a fault)
	h.updateFaults(allFaults{
		api.VideoLossAlarm:                true,
		api.VideoTamperingAlarm:           true,
		api.CameraRecordingExceptionAlarm: true,
		api.CameraRecordingRecovered:      true, // Not a fault, just a marker
	})

	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults := checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(3, len(faults)); diff != "" {
		t.Errorf("initial fault count mismatch (-want +got):\n%s", diff)
	}

	// Simulate driver reboot by clearing all faults
	h.updateFaults(allFaults{})

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(0, len(faults)); diff != "" {
		t.Errorf("fault count after reboot and clear mismatch (-want +got):\n%s", diff)
	}

	// Add back some faults
	h.updateFaults(allFaults{
		api.VideoLossAlarm:      true,
		api.VideoTamperingAlarm: true,
	})

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 2)

	// Partially clear
	h.updateFaults(allFaults{
		api.VideoLossAlarm: true,
	})

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(1, len(faults)); diff != "" {
		t.Errorf("fault count after reboot with partial faults mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(api.VideoLossAlarm, faults[0].Code.Code); diff != "" {
		t.Errorf("remaining fault code mismatch (-want +got):\n%s", diff)
	}
}

func TestStatusToHealthCode(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedCode   string
		expectedSystem string
	}{
		{
			name:           "VideoLossAlarm",
			status:         api.VideoLossAlarm,
			expectedCode:   api.VideoLossAlarm,
			expectedSystem: SystemName,
		},
		{
			name:           "VideoTamperingAlarm",
			status:         api.VideoTamperingAlarm,
			expectedCode:   api.VideoTamperingAlarm,
			expectedSystem: SystemName,
		},
		{
			name:           "CameraRecordingExceptionAlarm",
			status:         api.CameraRecordingExceptionAlarm,
			expectedCode:   api.CameraRecordingExceptionAlarm,
			expectedSystem: SystemName,
		},
		{
			name:           "CameraRecordingRecovered",
			status:         api.CameraRecordingRecovered,
			expectedCode:   api.CameraRecordingRecovered,
			expectedSystem: SystemName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := statusToHealthCode(tt.status)

			if diff := cmp.Diff(tt.expectedCode, code.Code); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.expectedSystem, code.System); diff != "" {
				t.Errorf("system mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAllFaultsHasFault(t *testing.T) {
	tests := []struct {
		name     string
		faults   allFaults
		hasFault bool
	}{
		{
			name:     "no faults",
			faults:   allFaults{},
			hasFault: false,
		},
		{
			name: "video loss",
			faults: allFaults{
				api.VideoLossAlarm: true,
			},
			hasFault: true,
		},
		{
			name: "video tampering",
			faults: allFaults{
				api.VideoTamperingAlarm: true,
			},
			hasFault: true,
		},
		{
			name: "recording exception only",
			faults: allFaults{
				api.CameraRecordingExceptionAlarm: true,
			},
			hasFault: true,
		},
		{
			name: "recording recovered only - not a fault",
			faults: allFaults{
				api.CameraRecordingRecovered: true,
			},
			hasFault: false, // Recovered is not considered a fault
		},
		{
			name: "recording exception and recovered",
			faults: allFaults{
				api.CameraRecordingExceptionAlarm: true,
				api.CameraRecordingRecovered:      true,
			},
			hasFault: true, // hasFault() checks CameraRecordingExceptionAlarm directly, ignores recovered
		},
		{
			name: "multiple faults with recovered",
			faults: allFaults{
				api.VideoLossAlarm:                true,
				api.CameraRecordingExceptionAlarm: true,
				api.CameraRecordingRecovered:      true,
			},
			hasFault: true, // VideoLossAlarm OR CameraRecordingExceptionAlarm makes this true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.faults.hasFault()
			if diff := cmp.Diff(tt.hasFault, result); diff != "" {
				t.Errorf("hasFault result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateReliability(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedState      gen.HealthCheck_Reliability_State
		expectedErrorCode  string
		expectedSummary    string
		expectErrorPresent bool
	}{
		{
			name:               "no error - reliable",
			err:                nil,
			expectedState:      gen.HealthCheck_Reliability_RELIABLE,
			expectErrorPresent: false,
		},
		{
			name:               "generic error - no response",
			err:                errors.New("connection timeout"),
			expectedState:      gen.HealthCheck_Reliability_NO_RESPONSE,
			expectedErrorCode:  Offline,
			expectedSummary:    "Device Offline",
			expectErrorPresent: true,
		},
		{
			name:               "context deadline exceeded - no response",
			err:                context.DeadlineExceeded,
			expectedState:      gen.HealthCheck_Reliability_NO_RESPONSE,
			expectedErrorCode:  Offline,
			expectedSummary:    "Device Offline",
			expectErrorPresent: true,
		},
		{
			name:               "json unmarshal type error - bad response",
			err:                &json.UnmarshalTypeError{Value: "string", Type: nil},
			expectedState:      gen.HealthCheck_Reliability_BAD_RESPONSE,
			expectedErrorCode:  BadResponse,
			expectedSummary:    "Bad Response",
			expectErrorPresent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupTestHarness(t)
			h.updateReliability(tt.err)

			checks := h.getHealthChecks(t)
			require.Len(t, checks, 1)

			reliability := checks[0].GetReliability()
			require.NotNil(t, reliability)

			if diff := cmp.Diff(tt.expectedState, reliability.State, protocmp.Transform()); diff != "" {
				t.Errorf("reliability state mismatch (-want +got):\n%s", diff)
			}

			if tt.expectErrorPresent {
				require.NotNil(t, reliability.LastError, "expected LastError to be present")

				if tt.expectedErrorCode != "" {
					require.NotNil(t, reliability.LastError.Code)
					if diff := cmp.Diff(tt.expectedErrorCode, reliability.LastError.Code.Code); diff != "" {
						t.Errorf("error code mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(SystemName, reliability.LastError.Code.System); diff != "" {
						t.Errorf("system name mismatch (-want +got):\n%s", diff)
					}
				}

				if tt.expectedSummary != "" {
					if diff := cmp.Diff(tt.expectedSummary, reliability.LastError.SummaryText); diff != "" {
						t.Errorf("error summary mismatch (-want +got):\n%s", diff)
					}
				}
			} else {
				require.Nil(t, reliability.LastError, "expected LastError to be nil")
			}
		})
	}
}

func TestReliabilityTransitions(t *testing.T) {
	h := setupTestHarness(t)

	// Start with reliable state
	h.updateReliability(nil)
	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, gen.HealthCheck_Reliability_RELIABLE, checks[0].GetReliability().State)

	// Transition to no response
	h.updateReliability(errors.New("network error"))
	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, gen.HealthCheck_Reliability_NO_RESPONSE, checks[0].GetReliability().State)
	require.NotNil(t, checks[0].GetReliability().LastError)

	// Transition to bad response
	h.updateReliability(&json.UnmarshalTypeError{Value: "test"})
	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, gen.HealthCheck_Reliability_BAD_RESPONSE, checks[0].GetReliability().State)
	require.Equal(t, BadResponse, checks[0].GetReliability().LastError.Code.Code)

	// Recover to reliable
	h.updateReliability(nil)
	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, gen.HealthCheck_Reliability_RELIABLE, checks[0].GetReliability().State)
	// Note: LastError is preserved to show what the last error was, even when state is now RELIABLE
	require.NotNil(t, checks[0].GetReliability().LastError)
}

func TestReliabilityWithFaults(t *testing.T) {
	h := setupTestHarness(t)

	// Set reliable state
	h.updateReliability(nil)

	// Add some faults
	h.updateFaults(allFaults{
		api.VideoLossAlarm:      true,
		api.VideoTamperingAlarm: true,
	})

	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)

	// Should have reliable state
	require.Equal(t, gen.HealthCheck_Reliability_RELIABLE, checks[0].GetReliability().State)

	// Should have faults
	faults := checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 2)

	// Should be abnormal due to faults
	require.Equal(t, gen.HealthCheck_ABNORMAL, checks[0].Normality)

	// Now set unreliable state
	h.updateReliability(errors.New("connection lost"))

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)

	// Should have no response state
	require.Equal(t, gen.HealthCheck_Reliability_NO_RESPONSE, checks[0].GetReliability().State)

	// Faults should still be present
	faults = checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 2)

	// Should still be abnormal
	require.Equal(t, gen.HealthCheck_ABNORMAL, checks[0].Normality)
}

type devicesServerModel struct {
	devices.Collection
}

func (m devicesServerModel) ClientConn() grpc.ClientConnInterface {
	return nil
}
