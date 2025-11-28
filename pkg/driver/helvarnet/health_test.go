package helvarnet

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func setup(devs *devicespb.Collection) *healthpb.Registry {

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
	deviceName := "helvarnet-device-1"
	reg := setup(devs)
	healthChecks := reg.ForOwner("example")

	_, _ = devs.Update(&gen.Device{Name: deviceName}, resource.WithCreateIfAbsent())

	check := getDeviceHealthCheck()
	fc, err := healthChecks.NewFaultCheck(deviceName, check)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)

	return &testHarness{
		devs:   devs,
		client: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, server)),
		fc:     fc,
		ctx:    context.Background(),
	}
}

func (h *testHarness) updateStatus(status int64) {
	updateDeviceFaults(h.ctx, status, h.fc)
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

func TestHelvarnetFaults(t *testing.T) {
	tests := []struct {
		name          string
		status        int64
		expectedCount int
		expectedCodes []string
	}{
		{
			name:          "no errors",
			status:        0,
			expectedCount: 0,
			expectedCodes: nil,
		},
		{
			name:          "single fault",
			status:        0x00000001,
			expectedCount: 1,
			expectedCodes: []string{strconv.Itoa(0x00000001)},
		},
		{
			name:          "double fault",
			status:        0x00000011,
			expectedCount: 2,
			expectedCodes: []string{strconv.Itoa(0x00000001), strconv.Itoa(0x00000010)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupTestHarness(t)
			h.updateStatus(tt.status)

			checks := h.getHealthChecks(t)
			require.Len(t, checks, 1)

			faults := checks[0].GetFaults().GetCurrentFaults()
			if diff := cmp.Diff(tt.expectedCount, len(faults)); diff != "" {
				t.Errorf("unexpected fault count (-want +got):\n%s", diff)
			}

			for i, expectedCode := range tt.expectedCodes {
				if diff := cmp.Diff(expectedCode, faults[i].Code.Code); diff != "" {
					t.Errorf("fault[%d].Code.Code mismatch (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}

func TestFaultLifecycle(t *testing.T) {
	tests := []struct {
		name  string
		steps []struct {
			status       int64
			faultCount   int
			normality    gen.HealthCheck_Normality
			expectedCode string // only for single fault cases
		}
	}{
		{
			name: "add multiple faults then clear all",
			steps: []struct {
				status       int64
				faultCount   int
				normality    gen.HealthCheck_Normality
				expectedCode string
			}{
				{
					status:     0x00000001 | 0x00000002 | 0x00000004, // Disabled | LampFailure | Missing
					faultCount: 3,
					normality:  gen.HealthCheck_ABNORMAL,
				},
				{
					status:     0,
					faultCount: 0,
					normality:  gen.HealthCheck_NORMAL,
				},
			},
		},
		{
			name: "add multiple faults then partial clear",
			steps: []struct {
				status       int64
				faultCount   int
				normality    gen.HealthCheck_Normality
				expectedCode string
			}{
				{
					status:     0x00000001 | 0x00000002 | 0x00000004,
					faultCount: 3,
					normality:  gen.HealthCheck_ABNORMAL,
				},
				{
					status:       0x00000001, // Keep only Disabled
					faultCount:   1,
					normality:    gen.HealthCheck_ABNORMAL,
					expectedCode: strconv.Itoa(0x00000001),
				},
				{
					status:     0,
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
				h.updateStatus(step.status)

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

func TestFaultUpdate(t *testing.T) {
	h := setupTestHarness(t)

	// Step 1: Add initial fault
	h.updateStatus(0x00000001) // Disabled

	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults := checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(1, len(faults)); diff != "" {
		t.Errorf("initial fault count mismatch (-want +got):\n%s", diff)
	}

	// Step 2: Update with same fault code (should update, not duplicate)
	h.updateStatus(0x00000001) // Disabled again

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(1, len(faults)); diff != "" {
		t.Errorf("fault count after re-raising same fault (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(strconv.Itoa(0x00000001), faults[0].Code.Code); diff != "" {
		t.Errorf("fault code mismatch (-want +got):\n%s", diff)
	}

	// Step 3: Add a different fault (should now have 2 faults)
	h.updateStatus(0x00000001 | 0x00000002) // Disabled + LampFailure

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(2, len(faults)); diff != "" {
		t.Errorf("fault count after adding second fault (-want +got):\n%s", diff)
	}
}

func TestSpecialErrorCodes(t *testing.T) {
	tests := []struct {
		name                string
		status              int64
		expectedSummary     string
		expectedDetails     string
		expectedReliability gen.HealthCheck_Reliability_State
		expectedCode        string
	}{
		{
			name:                "device offline",
			status:              DeviceOfflineCode,
			expectedSummary:     "Device Offline",
			expectedDetails:     "No communication received from device since the last Smart Core restart",
			expectedReliability: gen.HealthCheck_Reliability_NO_RESPONSE,
			expectedCode:        strconv.Itoa(DeviceOfflineCode),
		},
		{
			name:                "bad response",
			status:              BadResponseCode,
			expectedSummary:     "Bad Response",
			expectedDetails:     "The device has sent an invalid response to a command",
			expectedReliability: gen.HealthCheck_Reliability_BAD_RESPONSE,
			expectedCode:        strconv.Itoa(BadResponseCode),
		},
		{
			name:                "unknown negative status",
			status:              -99,
			expectedSummary:     "Internal Driver Error",
			expectedDetails:     "The device has an unrecognised internal status code",
			expectedReliability: gen.HealthCheck_Reliability_UNRELIABLE,
			expectedCode:        strconv.Itoa(UnrecognisedErrorCode),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupTestHarness(t)

			// Set the error status
			h.updateStatus(tt.status)

			checks := h.getHealthChecks(t)
			require.Len(t, checks, 1)

			// Check reliability state
			if diff := cmp.Diff(tt.expectedReliability, checks[0].Reliability.State, protocmp.Transform()); diff != "" {
				t.Errorf("reliability state mismatch (-want +got):\n%s", diff)
			}

			// Check LastError contains the expected information
			lastError := checks[0].Reliability.LastError
			require.NotNil(t, lastError, "expected LastError to be set")

			if diff := cmp.Diff(tt.expectedSummary, lastError.SummaryText); diff != "" {
				t.Errorf("LastError summary mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.expectedDetails, lastError.DetailsText); diff != "" {
				t.Errorf("LastError details mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.expectedCode, lastError.Code.Code); diff != "" {
				t.Errorf("LastError code mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(SystemName, lastError.Code.System); diff != "" {
				t.Errorf("LastError system mismatch (-want +got):\n%s", diff)
			}

			// Clear the error
			h.updateStatus(0)

			checks = h.getHealthChecks(t)
			require.Len(t, checks, 1)

			// After clearing, there should be no faults
			faults := checks[0].GetFaults().GetCurrentFaults()
			if diff := cmp.Diff(0, len(faults)); diff != "" {
				t.Errorf("fault count after clear mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDriverRebootClearsFaults(t *testing.T) {
	h := setupTestHarness(t)

	h.updateStatus(0x00000001 | 0x00000002 | 0x00000004)

	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults := checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(3, len(faults)); diff != "" {
		t.Errorf("initial fault count mismatch (-want +got):\n%s", diff)
	}

	h.updateStatus(0)

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(0, len(faults)); diff != "" {
		t.Errorf("fault count after reboot and clear mismatch (-want +got):\n%s", diff)
	}

	h.updateStatus(0x00000001 | 0x00000002)
	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 2)

	h.updateStatus(0x00000001)

	checks = h.getHealthChecks(t)
	require.Len(t, checks, 1)
	faults = checks[0].GetFaults().GetCurrentFaults()
	if diff := cmp.Diff(1, len(faults)); diff != "" {
		t.Errorf("fault count after reboot with partial faults mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(strconv.Itoa(0x00000001), faults[0].Code.Code); diff != "" {
		t.Errorf("remaining fault code mismatch (-want +got):\n%s", diff)
	}
}

type devicesServerModel struct {
	devices.Collection
}

func (m devicesServerModel) ClientConn() grpc.ClientConnInterface {
	return nil
}
