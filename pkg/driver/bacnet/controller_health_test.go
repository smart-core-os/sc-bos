package bacnet

import (
	"context"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func newTestControllerHealth(t *testing.T, threshold int) (*healthpb.Registry, *controllerHealth) {
	t.Helper()
	r := healthpb.NewRegistry()
	checks := r.ForOwner("test-owner")
	fc, err := checks.NewFaultCheck("test-controller", createControllerHealthCheck(
		healthpb.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED,
		healthpb.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED,
	))
	if err != nil {
		t.Fatalf("NewFaultCheck: %v", err)
	}
	return r, newControllerHealth(fc, threshold)
}

func controllerReliabilityState(r *healthpb.Registry) healthpb.HealthCheck_Reliability_State {
	c := r.GetCheck("test-controller", healthpb.AbsID("test-owner", "controllerStatusCheck"))
	if c == nil {
		return healthpb.HealthCheck_Reliability_STATE_UNSPECIFIED
	}
	return c.GetReliability().GetState()
}

func isUnhealthy(r *healthpb.Registry) bool {
	s := controllerReliabilityState(r)
	return s != healthpb.HealthCheck_Reliability_STATE_UNSPECIFIED &&
		s != healthpb.HealthCheck_Reliability_RELIABLE
}

func TestControllerHealth_TwoDevices_50Threshold(t *testing.T) {
	ctx := context.Background()
	r, ch := newTestControllerHealth(t, 50)
	ch.register("dev1")
	ch.register("dev2")

	// both failing → unhealthy (2/2 = 100% >= 50%)
	ch.setFailing(ctx, "dev1")
	ch.setFailing(ctx, "dev2")
	if !isUnhealthy(r) {
		t.Error("expected controller unhealthy when all devices fail")
	}

	// 1/2 = 50% meets threshold → still unhealthy
	ch.setOK(ctx, "dev1")
	if !isUnhealthy(r) {
		t.Error("expected controller still unhealthy at exactly 50% (1/2)")
	}

	// 0/2 = 0% → healthy
	ch.setOK(ctx, "dev2")
	if isUnhealthy(r) {
		t.Error("expected controller healthy when no devices fail")
	}
}

func TestControllerHealth_ThreeDevices_50Threshold(t *testing.T) {
	ctx := context.Background()
	r, ch := newTestControllerHealth(t, 50)
	ch.register("dev1")
	ch.register("dev2")
	ch.register("dev3")

	// 1/3 ≈ 33% < 50% → healthy
	ch.setFailing(ctx, "dev1")
	if isUnhealthy(r) {
		t.Error("expected controller healthy when 1/3 devices fail")
	}

	// 2/3 ≈ 67% >= 50% → unhealthy
	ch.setFailing(ctx, "dev2")
	if !isUnhealthy(r) {
		t.Error("expected controller unhealthy when 2/3 devices fail")
	}

	// recover one → 1/3 < 50% → healthy again
	ch.setOK(ctx, "dev2")
	if isUnhealthy(r) {
		t.Error("expected controller healthy after recovering to 1/3 failing")
	}
}

func TestControllerHealth_RegisterIdempotent(t *testing.T) {
	ctx := context.Background()
	_, ch := newTestControllerHealth(t, 50)
	ch.register("dev1")
	ch.register("dev1") // duplicate should not double-count

	ch.setFailing(ctx, "dev1")
	if len(ch.states) != 1 {
		t.Errorf("expected 1 device state entry, got %d", len(ch.states))
	}
}

func TestControllerHealth_100Threshold(t *testing.T) {
	ctx := context.Background()
	r, ch := newTestControllerHealth(t, 100)
	ch.register("dev1")
	ch.register("dev2")

	// 1/2 = 50% < 100% → healthy
	ch.setFailing(ctx, "dev1")
	if isUnhealthy(r) {
		t.Error("expected controller healthy when only 1/2 fail at 100% threshold")
	}

	// 2/2 = 100% >= 100% → unhealthy
	ch.setFailing(ctx, "dev2")
	if !isUnhealthy(r) {
		t.Error("expected controller unhealthy when all devices fail at 100% threshold")
	}
}
