package merge

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
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

func faultCount(get func() *healthpb.HealthCheck) int {
	return len(get().GetFaults().GetCurrentFaults())
}

var errBoom = errors.New("boom")

func TestTraitFault_Update(t *testing.T) {
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
		{"single real error", []error{errBoom}, true},
		{"nil and real error", []error{nil, errBoom}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc, get := newTestFaultCheck(t)
			newTraitFault(fc, 0).update("dev", trait.FanSpeed, tt.errs)
			faults := get().GetFaults().GetCurrentFaults()
			if gotFault := len(faults) > 0; gotFault != tt.wantFault {
				t.Errorf("got fault=%v (%d faults), want fault=%v", gotFault, len(faults), tt.wantFault)
			}
		})
	}
}

// TestTraitFault_Nil verifies a nil tracker, or one with a nil FaultCheck, is a
// no-op rather than a panic, even with real errors present.
func TestTraitFault_Nil(t *testing.T) {
	var f *traitFault
	f.update("dev", trait.FanSpeed, []error{errBoom})
	newTraitFault(nil, 0).update("dev", trait.FanSpeed, []error{errBoom})
}

// TestTraitFault_CountsOnlyNonNil verifies nil errors are excluded from the
// fault description and the reported error count.
func TestTraitFault_CountsOnlyNonNil(t *testing.T) {
	fc, get := newTestFaultCheck(t)
	newTraitFault(fc, 0).update("dev", trait.FanSpeed, []error{nil, errBoom, nil})

	faults := get().GetFaults().GetCurrentFaults()
	if len(faults) != 1 {
		t.Fatalf("want 1 fault, got %d", len(faults))
	}
	if got, want := faults[0].GetSummaryText(), "has 1 errors"; !strings.Contains(got, want) {
		t.Errorf("summary %q does not contain %q", got, want)
	}
}

// TestTraitFault_ClearsOnSuccess mirrors a fanSpeed poll recovering:
// a fault is set, then a subsequent successful poll (err == nil) clears it.
func TestTraitFault_ClearsOnSuccess(t *testing.T) {
	fc, get := newTestFaultCheck(t)
	f := newTraitFault(fc, 0)

	f.update("dev", trait.FanSpeed, []error{errBoom})
	if n := faultCount(get); n != 1 {
		t.Fatalf("setup: want 1 fault, got %d", n)
	}

	// A recovered poll: fanSpeed would have passed []error{nil} here pre-fix.
	f.update("dev", trait.FanSpeed, []error{nil})
	if n := faultCount(get); n != 0 {
		t.Errorf("want fault cleared, got %d faults", n)
	}
}

func TestTraitFault_Grace(t *testing.T) {
	const after = 2 * time.Minute
	const poll = 10 * time.Second

	t.Run("one failure", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			fc, get := newTestFaultCheck(t)
			f := newTraitFault(fc, after)
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			if n := faultCount(get); n != 0 {
				t.Errorf("want no fault after one failure, got %d", n)
			}
		})
	})

	t.Run("failures shorter than after", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			fc, get := newTestFaultCheck(t)
			f := newTraitFault(fc, after)
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			time.Sleep(after - time.Second)
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			if n := faultCount(get); n != 0 {
				t.Errorf("want no fault before %v of failures, got %d", after, n)
			}
		})
	})

	t.Run("failures reaching after", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			fc, get := newTestFaultCheck(t)
			f := newTraitFault(fc, after)
			start := time.Now()
			for time.Since(start) < after {
				f.update("dev", temperaturepb.TraitName, []error{errBoom})
				if n := faultCount(get); n != 0 {
					t.Fatalf("fault raised after %v of failures, want %v", time.Since(start), after)
				}
				time.Sleep(poll)
			}
			f.update("dev", temperaturepb.TraitName, []error{errBoom, errBoom})

			hc := get()
			if hc.GetNormality() != healthpb.HealthCheck_ABNORMAL {
				t.Errorf("normality = %v, want ABNORMAL", hc.GetNormality())
			}
			faults := hc.GetFaults().GetCurrentFaults()
			if len(faults) != 1 {
				t.Fatalf("want 1 fault, got %d", len(faults))
			}
			if got, want := faults[0].GetSummaryText(), "dev[smartcore.bos.Temperature] has 2 errors"; got != want {
				t.Errorf("summary = %q, want %q", got, want)
			}
			if got := faults[0].GetCode(); got.GetCode() != BacNetCommsError || got.GetSystem() != SystemName {
				t.Errorf("code = %v, want %q/%q", got, BacNetCommsError, SystemName)
			}
		})
	})

	// a clean poll ends the streak, so a device failing every other poll never faults: accepted, see traitFault
	t.Run("alternating failures", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			fc, get := newTestFaultCheck(t)
			f := newTraitFault(fc, after)
			for i := range 4 * int(after/poll) {
				var errs []error
				if i%2 == 0 {
					errs = []error{errBoom}
				}
				f.update("dev", temperaturepb.TraitName, errs)
				if n := faultCount(get); n != 0 {
					t.Fatalf("poll %d: want no fault, got %d", i, n)
				}
				time.Sleep(poll)
			}
		})
	})

	t.Run("recovery restarts the streak", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			fc, get := newTestFaultCheck(t)
			f := newTraitFault(fc, after)
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			time.Sleep(after)
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			if n := faultCount(get); n != 1 {
				t.Fatalf("setup: want 1 fault, got %d", n)
			}

			f.update("dev", temperaturepb.TraitName, nil)
			if n := faultCount(get); n != 0 {
				t.Fatalf("want fault cleared on a good read, got %d", n)
			}

			// A new failure only counts from now, not from the first streak.
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			time.Sleep(after - time.Second)
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			if n := faultCount(get); n != 0 {
				t.Errorf("want no fault before a new %v streak, got %d", after, n)
			}
			time.Sleep(time.Second)
			f.update("dev", temperaturepb.TraitName, []error{errBoom})
			if n := faultCount(get); n != 1 {
				t.Errorf("want fault after a new %v streak, got %d", after, n)
			}
		})
	})
}

// TestTraitFault_Concurrent mirrors the poll loop and Get calls polling at the
// same time. Run with -race.
func TestTraitFault_Concurrent(t *testing.T) {
	fc, get := newTestFaultCheck(t)
	f := newTraitFault(fc, 0)
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Go(func() {
			for i := range 100 {
				var errs []error
				if (g+i)%2 == 0 {
					errs = []error{errBoom}
				}
				f.update("dev", temperaturepb.TraitName, errs)
			}
		})
	}
	wg.Wait()

	f.update("dev", temperaturepb.TraitName, nil)
	if n := faultCount(get); n != 0 {
		t.Errorf("want fault cleared, got %d", n)
	}
}
