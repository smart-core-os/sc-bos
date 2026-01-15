package healthpb

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// FaultCheck updates a health check based on a general error value.
type FaultCheck struct {
	*checkBase
}

// newFaultCheck creates a new FaultCheck for the given health check.
func newFaultCheck(c *healthpb.HealthCheck) (*FaultCheck, error) {
	if err := normalizeFaultCheck(c); err != nil {
		return nil, err
	}
	return &FaultCheck{checkBase: &checkBase{check: c}}, nil
}

func normalizeFaultCheck(c *healthpb.HealthCheck) error {
	switch out := c.GetCheck().(type) {
	case nil:
		c.Check = &healthpb.HealthCheck_Faults_{Faults: &healthpb.HealthCheck_Faults{}}
		return nil
	case *healthpb.HealthCheck_Faults_:
		return nil
	default:
		return fmt.Errorf("check type must be faults, got %T", out)
	}
}

// SetFault sets the health check to have exactly the given fault.
// If err is nil, all faults are cleared.
// The health check state is updated to ABNORMAL if err is non-nil, or NORMAL if err is nil.
// The reliability is set to RELIABLE.
func (c *FaultCheck) SetFault(err *healthpb.HealthCheck_Error) {
	if err == nil {
		c.ClearFaults()
		return
	}
	c.writeFaults(func(old []*healthpb.HealthCheck_Error) []*healthpb.HealthCheck_Error {
		return []*healthpb.HealthCheck_Error{err}
	})
}

// AddOrUpdateFault adds or updates the given fault in the health check.
// If a fault with the same system and code (or summary text if system/code are not set) exists, it is replaced.
// Otherwise, the fault is added to the list.
// The health check state is updated to ABNORMAL and the reliability is set to RELIABLE.
func (c *FaultCheck) AddOrUpdateFault(err *healthpb.HealthCheck_Error) {
	c.writeFaults(func(old []*healthpb.HealthCheck_Error) []*healthpb.HealthCheck_Error {
		return addOrReplaceFault(old, err)
	})
}

// ClearFaults removes all faults from the health check.
// The health check state is updated to NORMAL.
// The reliability is set to RELIABLE.
func (c *FaultCheck) ClearFaults() {
	c.writeFaults(func(old []*healthpb.HealthCheck_Error) []*healthpb.HealthCheck_Error {
		return nil
	})
}

// RemoveFault removes the given fault from the health check.
// Faults are matched by their system and code, or summary text if that is not set.
// If the fault does not exist, no action is taken.
// If the fault is removed and no other faults remain, the health check state is updated to NORMAL.
// The reliability is set to RELIABLE.
func (c *FaultCheck) RemoveFault(err *healthpb.HealthCheck_Error) {
	if err == nil {
		return
	}
	c.writeFaults(func(old []*healthpb.HealthCheck_Error) []*healthpb.HealthCheck_Error {
		if len(old) == 0 {
			return old
		}
		i, found := findFault(err, old)
		if !found {
			return old
		}
		return slices.Delete(old, i, i+1)
	})
}

// addOrReplaceFault adds the new fault to the list, replacing any existing fault.
// Faults are matched by their system and code, or summary text if that is not set.
// The old slice must be sorted by code.system, code.code, summary_text.
func addOrReplaceFault(old []*healthpb.HealthCheck_Error, n *healthpb.HealthCheck_Error) []*healthpb.HealthCheck_Error {
	if n == nil {
		return old
	}
	if len(old) == 0 {
		return []*healthpb.HealthCheck_Error{n}
	}

	i, found := findFault(n, old)
	if found {
		old[i] = n
		return old
	}
	return slices.Insert(old, i, n)
}

func findFault(n *healthpb.HealthCheck_Error, faults []*healthpb.HealthCheck_Error) (int, bool) {
	return slices.BinarySearchFunc(faults, n, func(e *healthpb.HealthCheck_Error, t *healthpb.HealthCheck_Error) int {
		if e.GetCode() == nil && t.GetCode() == nil {
			return strings.Compare(e.GetSummaryText(), t.GetSummaryText())
		}
		if e.GetCode() == nil {
			return -1
		}
		if t.GetCode() == nil {
			return 1
		}
		// both codes are non-nil
		return cmp.Or(
			strings.Compare(e.GetCode().GetSystem(), t.GetCode().GetSystem()),
			strings.Compare(e.GetCode().GetCode(), t.GetCode().GetCode()),
		)
	})
}

func (c *FaultCheck) writeFaults(f func(old []*healthpb.HealthCheck_Error) []*healthpb.HealthCheck_Error) {
	c.write(func(dst *healthpb.HealthCheck) {
		out := dst.GetFaults()
		if out == nil {
			panic("no faults object, normalisation bypassed")
		}
		oldState := dst.GetNormality()
		oldFaults := out.GetCurrentFaults()
		newFaults := f(oldFaults)
		newState := healthpb.HealthCheck_NORMAL
		if len(newFaults) > 0 {
			newState = healthpb.HealthCheck_ABNORMAL
		}
		out.CurrentFaults = newFaults
		dst.Normality = newState
		updateStateTimes(dst, oldState, newState)
		// any error means the out is working, transport errors will call UpdateReliability directly
		makeReliable(dst)
	})
}
