package merge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

// newTestFaultCheck returns a FaultCheck and a getter for its current state.
func newTestFaultCheck(t *testing.T) (*healthpb.FaultCheck, func() *healthpb.HealthCheck) {
	t.Helper()
	reg := healthpb.NewRegistry()
	hc := &healthpb.HealthCheck{Id: "trait"}
	fc, err := reg.ForOwner("test").NewFaultCheck("dev", hc)
	if err != nil {
		t.Fatalf("NewFaultCheck: %v", err)
	}
	id := hc.Id // adjusted to the absolute id during NewFaultCheck
	return fc, func() *healthpb.HealthCheck {
		return reg.GetCheck("dev", id)
	}
}

func TestUpdateTraitFaultCheck(t *testing.T) {
	realErr := errors.New("boom")
	tests := []struct {
		name      string
		errs      []error
		wantFault bool
	}{
		{"nil slice", nil, false},
		{"empty slice", []error{}, false},
		// Regression: fanSpeed passed []error{err} even when err was nil, which
		// used to bypass the empty-check and panic on err.Error().
		{"single nil error", []error{nil}, false},
		{"multiple nil errors", []error{nil, nil}, false},
		{"single real error", []error{realErr}, true},
		{"nil and real error", []error{nil, realErr}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc, get := newTestFaultCheck(t)
			updateTraitFaultCheck(context.Background(), fc, "dev", trait.FanSpeed, tt.errs)
			faults := get().GetFaults().GetCurrentFaults()
			if gotFault := len(faults) > 0; gotFault != tt.wantFault {
				t.Errorf("got fault=%v (%d faults), want fault=%v", gotFault, len(faults), tt.wantFault)
			}
		})
	}
}

// TestUpdateTraitFaultCheck_NilFaultCheck verifies a nil FaultCheck is a no-op
// rather than a panic, even with real errors present.
func TestUpdateTraitFaultCheck_NilFaultCheck(t *testing.T) {
	updateTraitFaultCheck(context.Background(), nil, "dev", trait.FanSpeed, []error{errors.New("boom")})
}

// TestUpdateTraitFaultCheck_CountsOnlyNonNil verifies nil errors are excluded
// from the fault description and the reported error count.
func TestUpdateTraitFaultCheck_CountsOnlyNonNil(t *testing.T) {
	fc, get := newTestFaultCheck(t)
	errs := []error{nil, errors.New("boom"), nil}
	updateTraitFaultCheck(context.Background(), fc, "dev", trait.FanSpeed, errs)

	faults := get().GetFaults().GetCurrentFaults()
	if len(faults) != 1 {
		t.Fatalf("want 1 fault, got %d", len(faults))
	}
	if got, want := faults[0].GetSummaryText(), "has 1 errors"; !strings.Contains(got, want) {
		t.Errorf("summary %q does not contain %q", got, want)
	}
}

// TestUpdateTraitFaultCheck_ClearsOnSuccess mirrors a fanSpeed poll recovering:
// a fault is set, then a subsequent successful poll (err == nil) clears it.
func TestUpdateTraitFaultCheck_ClearsOnSuccess(t *testing.T) {
	fc, get := newTestFaultCheck(t)

	updateTraitFaultCheck(context.Background(), fc, "dev", trait.FanSpeed, []error{errors.New("boom")})
	if n := len(get().GetFaults().GetCurrentFaults()); n != 1 {
		t.Fatalf("setup: want 1 fault, got %d", n)
	}

	// A recovered poll: fanSpeed would have passed []error{nil} here pre-fix.
	updateTraitFaultCheck(context.Background(), fc, "dev", trait.FanSpeed, []error{nil})
	if n := len(get().GetFaults().GetCurrentFaults()); n != 0 {
		t.Errorf("want fault cleared, got %d faults", n)
	}
}
