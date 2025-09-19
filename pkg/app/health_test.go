package app

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vanti-dev/sc-bos/internal/node/nodeopts"
	"github.com/vanti-dev/sc-bos/pkg/app/sysconf"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/devicespb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb"
	"github.com/vanti-dev/sc-bos/pkg/node"
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
		Health: &sysconf.Health{
			DBPath: "db.sqlite3",
		},
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
	assertHealthChecks(t, dev.HealthChecks, want)
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

func assertHealthChecks(t *testing.T, got []*gen.HealthCheck, want ...*gen.HealthCheck) {
	t.Helper()
	if diff := cmp.Diff(want, got, protocmp.Transform(), protocmp.IgnoreFields(&gen.HealthCheck{}, "create_time")); diff != "" {
		t.Errorf("health checks mismatch (-want +got):\n%s", diff)
	}
}
