package app

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
)

func Test_setupHealthRegistry(t *testing.T) {
	t.Run("create check", func(t *testing.T) {
		h := newHealthTestHarness(t)
		h.createFaultCheck("dev1", "a test check")

		wantCheck := &gen.HealthCheck{
			Id:          healthpb.AbsID("test-owner", ""),
			DisplayName: "a test check",
			CreateTime:  timestamppb.Now(),
			Check:       &gen.HealthCheck_Faults_{Faults: &gen.HealthCheck_Faults{}},
		}

		h.assertCheckInHealthApi(t, "dev1", wantCheck)
		h.assertCheckInDevicesApi(t, "dev1", wantCheck)
	})

	t.Run("update check", func(t *testing.T) {
		h := newHealthTestHarness(t)
		check := h.createFaultCheck("dev1", "updatable check")

		check.SetFault(&gen.HealthCheck_Error{SummaryText: "test error"})
		h.assertCheckNormality(t, "dev1", gen.HealthCheck_ABNORMAL)
		h.assertCheckHasFault(t, "dev1", "test error")

		check.ClearFaults()
		h.assertCheckNormality(t, "dev1", gen.HealthCheck_NORMAL)
		h.assertCheckHasNoFaults(t, "dev1")
	})

	t.Run("delete check", func(t *testing.T) {
		h := newHealthTestHarness(t)
		check := h.createFaultCheck("dev1", "deletable check")

		h.assertCheckExists(t, "dev1")
		check.Dispose()
		h.assertCheckDeleted(t, "dev1")
	})

	t.Run("seed from history", func(t *testing.T) {
		const (
			deviceName   = "dev2"
			faultSummary = "historical error"
			faultDetails = "this error should be restored"
		)
		h := newHealthTestHarness(t)

		check1 := h.createFaultCheck(deviceName, "seeded check")
		check1.SetFault(&gen.HealthCheck_Error{
			SummaryText: faultSummary,
			DetailsText: faultDetails,
		})

		oldCheck := h.getOnlyCheck(t, deviceName)
		oldNormality := oldCheck.Normality

		check1.Dispose()
		h.assertCheckDeleted(t, deviceName)

		// Create a new check with the same ID - it should be seeded from history
		check2 := h.createFaultCheck(deviceName, "seeded check")
		defer check2.Dispose()

		h.assertCheckNormality(t, deviceName, oldNormality)
		h.assertCheckHasFaultWithDetails(t, deviceName, faultSummary, faultDetails)
	})

	t.Run("value filtering", func(t *testing.T) {
		t.Run("measured values in HealthApi", func(t *testing.T) {
			h := newHealthTestHarness(t)
			check := h.createBoundsCheck("dev3", "temperature check", 25.5)
			defer check.Dispose()
			h.assertCheckHasCurrentValue(t, "dev3", 25.5)
		})

		t.Run("measured values omitted from DevicesApi", func(t *testing.T) {
			h := newHealthTestHarness(t)
			check := h.createBoundsCheck("dev4", "humidity check", 60.0)
			defer check.Dispose()
			h.assertDeviceCheckHasNoCurrentValue(t, "dev4")
		})

		t.Run("update measured value", func(t *testing.T) {
			h := newHealthTestHarness(t)
			check := h.createBoundsCheck("dev5", "pressure check", 100.0)
			defer check.Dispose()

			check.UpdateValue(h.ctx, healthpb.FloatValue(120.0))
			h.assertCheckHasCurrentValue(t, "dev5", 120.0)
			h.assertDeviceCheckHasNoCurrentValue(t, "dev5")
		})
	})
}

func Test_removeMeasuredValues(t *testing.T) {
	tests := []struct {
		name         string
		input        *gen.HealthCheck
		wantVal      *gen.HealthCheck
		wantNoChange bool
	}{
		{
			name:         "nil check",
			input:        nil,
			wantNoChange: true,
		},
		{
			name: "fault check unchanged",
			input: &gen.HealthCheck{
				Id:          "test-id",
				DisplayName: "test check",
				Check: &gen.HealthCheck_Faults_{
					Faults: &gen.HealthCheck_Faults{
						CurrentFaults: []*gen.HealthCheck_Error{
							{SummaryText: "test error"},
						},
					},
				},
			},
			wantNoChange: true,
		},
		{
			name: "bounds check without current_value unchanged",
			input: &gen.HealthCheck{
				Id:          "test-id",
				DisplayName: "test check",
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: &gen.HealthCheck_Bounds_NormalValue{
							NormalValue: healthpb.FloatValue(20.0),
						},
						DisplayUnit: "°C",
					},
				},
			},
			wantNoChange: true,
		},
		{
			name: "bounds check with current_value removed",
			input: &gen.HealthCheck{
				Id:          "temp-check",
				DisplayName: "Room Temperature",
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						CurrentValue: healthpb.FloatValue(21.5),
						Expected: &gen.HealthCheck_Bounds_NormalRange{
							NormalRange: &gen.HealthCheck_ValueRange{
								Low:  healthpb.FloatValue(18.0),
								High: healthpb.FloatValue(24.0),
							},
						},
						DisplayUnit: "°C",
					},
				},
			},
			wantVal: &gen.HealthCheck{
				Id:          "temp-check",
				DisplayName: "Room Temperature",
				Check: &gen.HealthCheck_Bounds_{
					Bounds: &gen.HealthCheck_Bounds{
						Expected: &gen.HealthCheck_Bounds_NormalRange{
							NormalRange: &gen.HealthCheck_ValueRange{
								Low:  healthpb.FloatValue(18.0),
								High: healthpb.FloatValue(24.0),
							},
						},
						DisplayUnit: "°C",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTransform := proto.Clone(tt.input)
			got := removeMeasuredValues(tt.input)
			want := tt.wantVal
			if tt.wantNoChange {
				want = tt.input
			}

			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("removeMeasuredValues() mismatch (-want +got):\n%s", diff)
			}
			// make sure input was not modified
			if diff := cmp.Diff(beforeTransform, tt.input, protocmp.Transform()); diff != "" {
				t.Errorf("input was modified (-want +got):\n%s", diff)
			}
		})
	}
}

// healthTestHarness provides a convenient interface for testing health registry functionality
type healthTestHarness struct {
	ctx             context.Context
	t               *testing.T
	registry        *healthpb.Registry
	devices         *devicespb.Collection
	healthApiClient gen.HealthApiClient
	owner           string
}

func newHealthTestHarness(t *testing.T) *healthTestHarness {
	ctx := t.Context()
	cfg := sysconf.Config{
		DataDir: t.TempDir(),
		Health:  &sysconf.Health{},
	}
	devices := devicespb.NewCollection()
	announcer := node.New("test-node", nodeopts.WithStore(devices))
	healthApiClient := gen.NewHealthApiClient(announcer.ClientConn())

	r, dispose, err := setupHealthRegistry(ctx, cfg, devices, announcer, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("setupHealthRegistry() error = %v", err)
	}
	t.Cleanup(func() {
		if err := dispose(); err != nil {
			t.Errorf("dispose() error = %v", err)
		}
	})

	devs := devices.ListDevices()
	if len(devs) != 1 { // node itself
		t.Fatalf("expected 1 device in device store, got %d", len(devs))
	}

	return &healthTestHarness{
		ctx:             ctx,
		t:               t,
		registry:        r,
		devices:         devices,
		healthApiClient: healthApiClient,
		owner:           "test-owner",
	}
}

func (h *healthTestHarness) createFaultCheck(deviceName, displayName string) *healthpb.FaultCheck {
	check, err := h.registry.ForOwner(h.owner).NewFaultCheck(deviceName, &gen.HealthCheck{
		DisplayName: displayName,
	})
	if err != nil {
		h.t.Fatalf("NewFaultCheck() error = %v", err)
	}
	return check
}

func (h *healthTestHarness) createBoundsCheck(deviceName, displayName string, currentValue float64) *healthpb.BoundsCheck {
	check, err := h.registry.ForOwner(h.owner).NewBoundsCheck(deviceName, &gen.HealthCheck{
		DisplayName: displayName,
		Check: &gen.HealthCheck_Bounds_{
			Bounds: &gen.HealthCheck_Bounds{
				Expected: &gen.HealthCheck_Bounds_NormalRange{
					NormalRange: &gen.HealthCheck_ValueRange{
						Low:  healthpb.FloatValue(0.0),
						High: healthpb.FloatValue(100.0),
					},
				},
			},
		},
	})
	if err != nil {
		h.t.Fatalf("NewBoundsCheck() error = %v", err)
	}
	// Set initial value
	check.UpdateValue(h.ctx, healthpb.FloatValue(currentValue))
	return check
}

func (h *healthTestHarness) getOnlyCheck(t *testing.T, deviceName string) *gen.HealthCheck {
	t.Helper()
	apiChecks, err := h.healthApiClient.ListHealthChecks(h.ctx, &gen.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		t.Fatalf("ListHealthChecks() error = %v", err)
	}
	if len(apiChecks.HealthChecks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(apiChecks.HealthChecks))
	}
	return apiChecks.HealthChecks[0]
}

func (h *healthTestHarness) assertCheckInHealthApi(t *testing.T, deviceName string, want *gen.HealthCheck) {
	t.Helper()
	apiChecks, err := h.healthApiClient.ListHealthChecks(h.ctx, &gen.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		t.Fatalf("ListHealthChecks() error = %v", err)
	}
	assertHealthChecks(t, apiChecks.HealthChecks, want)
}

func (h *healthTestHarness) assertCheckInDevicesApi(t *testing.T, deviceName string, want *gen.HealthCheck) {
	t.Helper()
	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if dev == nil {
		t.Fatalf("GetDevice() returned nil device")
	}
	// Device API should omit measured values
	wantWithoutMeasured := removeMeasuredValues(want)
	assertHealthChecks(t, dev.HealthChecks, wantWithoutMeasured)
}

func (h *healthTestHarness) assertCheckNormality(t *testing.T, deviceName string, want gen.HealthCheck_Normality) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	if gotCheck.Normality != want {
		t.Errorf("expected normality %v, got %v", want, gotCheck.Normality)
	}

	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if dev == nil {
		t.Fatalf("GetDevice() returned nil device")
	}
	if len(dev.HealthChecks) > 0 && dev.HealthChecks[0].Normality != want {
		t.Errorf("expected normality %v in device, got %v", want, dev.HealthChecks[0].Normality)
	}
}

func (h *healthTestHarness) assertCheckHasFault(t *testing.T, deviceName, faultSummary string) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	faults := gotCheck.GetFaults().GetCurrentFaults()
	if len(faults) != 1 {
		t.Fatalf("expected 1 fault, got %d", len(faults))
	}
	if faults[0].SummaryText != faultSummary {
		t.Errorf("expected fault summary %q, got %q", faultSummary, faults[0].SummaryText)
	}
}

func (h *healthTestHarness) assertCheckHasFaultWithDetails(t *testing.T, deviceName, faultSummary, faultDetails string) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	faults := gotCheck.GetFaults().GetCurrentFaults()
	if len(faults) != 1 {
		t.Fatalf("expected 1 fault, got %d", len(faults))
	}
	if faults[0].SummaryText != faultSummary {
		t.Errorf("expected fault summary %q, got %q", faultSummary, faults[0].SummaryText)
	}
	if faults[0].DetailsText != faultDetails {
		t.Errorf("expected fault details %q, got %q", faultDetails, faults[0].DetailsText)
	}
}

func (h *healthTestHarness) assertCheckHasNoFaults(t *testing.T, deviceName string) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	faults := gotCheck.GetFaults().GetCurrentFaults()
	if len(faults) != 0 {
		t.Errorf("expected 0 faults, got %d", len(faults))
	}
}

func (h *healthTestHarness) assertCheckExists(t *testing.T, deviceName string) {
	t.Helper()
	_, err := h.healthApiClient.ListHealthChecks(h.ctx, &gen.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		t.Fatalf("expected check to exist, but got error: %v", err)
	}
}

func (h *healthTestHarness) assertCheckDeleted(t *testing.T, deviceName string) {
	t.Helper()
	apiChecks, err := h.healthApiClient.ListHealthChecks(h.ctx, &gen.ListHealthChecksRequest{Name: deviceName})
	if err != nil {
		if status.Code(err) != codes.NotFound {
			t.Fatalf("unexpected error from ListHealthChecks() = %v", err)
		}
	} else if len(apiChecks.HealthChecks) != 0 {
		t.Errorf("expected 0 checks after disposal, got %d", len(apiChecks.HealthChecks))
	}

	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			t.Fatalf("GetDevice() error = %v", err)
		}
	} else {
		if dev == nil {
			t.Fatalf("GetDevice() returned nil device")
		}
		if len(dev.HealthChecks) != 0 {
			t.Errorf("expected 0 checks in device after disposal, got %d", len(dev.HealthChecks))
		}
	}
}

func (h *healthTestHarness) assertCheckHasCurrentValue(t *testing.T, deviceName string, expectedValue float64) {
	t.Helper()
	gotCheck := h.getOnlyCheck(t, deviceName)
	bounds := gotCheck.GetBounds()
	if bounds == nil {
		t.Fatalf("expected bounds check, got %T", gotCheck.GetCheck())
	}
	currentValue := bounds.GetCurrentValue()
	if currentValue == nil {
		t.Fatalf("expected current_value to be set, got nil")
	}
	gotValue := currentValue.GetFloatValue()
	if gotValue != expectedValue {
		t.Errorf("expected current_value %v, got %v", expectedValue, gotValue)
	}
}

func (h *healthTestHarness) assertDeviceCheckHasNoCurrentValue(t *testing.T, deviceName string) {
	t.Helper()
	dev, err := h.devices.GetDevice(deviceName)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if dev == nil {
		t.Fatalf("GetDevice() returned nil device")
	}
	if len(dev.HealthChecks) == 0 {
		t.Fatalf("expected at least 1 check in device, got 0")
	}
	bounds := dev.HealthChecks[0].GetBounds()
	if bounds == nil {
		t.Fatalf("expected bounds check, got %T", dev.HealthChecks[0].GetCheck())
	}
	if bounds.GetCurrentValue() != nil {
		t.Errorf("expected current_value to be nil in device API, got %v", bounds.GetCurrentValue())
	}
}

func assertHealthChecks(t *testing.T, got []*gen.HealthCheck, want ...*gen.HealthCheck) {
	t.Helper()
	if diff := cmp.Diff(want, got, protocmp.Transform(), protocmp.IgnoreFields(&gen.HealthCheck{}, "create_time")); diff != "" {
		t.Errorf("health checks mismatch (-want +got):\n%s", diff)
	}
}
