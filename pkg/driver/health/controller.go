package health

import (
	"context"
	"fmt"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

const controllerUnhealthy = "ControllerUnhealthy"

type healthState int

const (
	stateUnknown healthState = iota
	stateFailing
	stateOK
)

// ControllerHealth aggregates per-device health states for a single controller (identified by IP address).
// When the proportion of failing devices reaches the threshold, the controller's FaultCheck is marked unhealthy.
type ControllerHealth struct {
	faultCheck *healthpb.FaultCheck
	threshold  int    // [0,100]: % of failing devices that triggers unhealthy
	systemName string // used in error code System field

	mu     sync.Mutex
	states map[string]healthState // device name → current state
}

func NewControllerHealth(fc *healthpb.FaultCheck, threshold int, systemName string) *ControllerHealth {
	return &ControllerHealth{
		faultCheck: fc,
		threshold:  threshold,
		systemName: systemName,
		states:     make(map[string]healthState),
	}
}

// Dispose releases the underlying FaultCheck.
func (c *ControllerHealth) Dispose() {
	c.faultCheck.Dispose()
}

// Register adds a device to controller tracking. Idempotent — safe to call multiple times.
func (c *ControllerHealth) Register(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.states[name]; !exists {
		c.states[name] = stateUnknown
	}
}

// SetFailing marks the named device as failing and recalculates controller health.
func (c *ControllerHealth) SetFailing(ctx context.Context, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.states[name] == stateFailing {
		return
	}
	c.states[name] = stateFailing
	c.recalculate(ctx)
}

// SetOK marks the named device as healthy and recalculates controller health.
func (c *ControllerHealth) SetOK(ctx context.Context, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.states[name] == stateOK {
		return
	}
	c.states[name] = stateOK
	c.recalculate(ctx)
}

// recalculate must be called with c.mu held.
func (c *ControllerHealth) recalculate(ctx context.Context) {
	total := len(c.states)
	if total == 0 {
		return
	}
	failingCount := 0
	for _, state := range c.states {
		if state == stateFailing {
			failingCount++
		}
	}
	code := &healthpb.HealthCheck_Error_Code{
		Code:   controllerUnhealthy,
		System: c.systemName,
	}
	if failingCount*100/total >= c.threshold {
		c.faultCheck.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
			State: healthpb.HealthCheck_Reliability_CONN_TRANSIENT_FAILURE,
			LastError: &healthpb.HealthCheck_Error{
				SummaryText: fmt.Sprintf("%d/%d devices on controller are failing", failingCount, total),
				Code:        code,
			},
		})
	} else {
		c.faultCheck.RemoveFault(&healthpb.HealthCheck_Error{Code: code})
	}
}
