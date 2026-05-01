package bacnet

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/merge"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const (
	DeviceUnreachable   = "DeviceUnreachable"
	ControllerUnhealthy = "ControllerUnhealthy"
)

func createDeviceHealthCheck(occupant healthpb.HealthCheck_OccupantImpact, equipment healthpb.HealthCheck_EquipmentImpact) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              "deviceStatusCheck",
		DisplayName:     "Device Status Check",
		Description:     "Checks the device is reachable and responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func createTraitHealthCheck(t trait.Name, occupant healthpb.HealthCheck_OccupantImpact, equipment healthpb.HealthCheck_EquipmentImpact) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              fmt.Sprintf("%s traitStatusCheck", t.String()),
		DisplayName:     fmt.Sprintf("%s Trait Status Check", t.String()),
		Description:     fmt.Sprintf("Checks %s is working correctly", t.String()),
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func updateRequestErrorStatus(ctx context.Context, deviceHealth *healthpb.FaultCheck, name, request string, err error) {
	if deviceHealth == nil {
		return
	}
	problemName := fmt.Sprintf("%s.%s", name, "requested")
	if errors.Is(err, context.DeadlineExceeded) {
		deviceHealth.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
			State: healthpb.HealthCheck_Reliability_CONN_TRANSIENT_FAILURE,
			LastError: &healthpb.HealthCheck_Error{
				SummaryText: "Device request timed out",
				DetailsText: fmt.Sprintf("%s %s: %v", problemName, request, err),
				Code:        statusToHealthCode(DeviceUnreachable),
			},
		})
		return
	}
	if err != nil {
		deviceHealth.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
			State: healthpb.HealthCheck_Reliability_BAD_RESPONSE,
			LastError: &healthpb.HealthCheck_Error{
				SummaryText: "Device request error",
				DetailsText: fmt.Sprintf("%s %s: %v", problemName, request, err),
				Code:        statusToHealthCode(DeviceUnreachable),
			},
		})
		return
	}

	deviceHealth.RemoveFault(&healthpb.HealthCheck_Error{
		Code: statusToHealthCode(DeviceUnreachable),
	})
}

func statusToHealthCode(code string) *healthpb.HealthCheck_Error_Code {
	return &healthpb.HealthCheck_Error_Code{
		Code:   code,
		System: merge.SystemName,
	}
}

func createControllerHealthCheck(occupant healthpb.HealthCheck_OccupantImpact, equipment healthpb.HealthCheck_EquipmentImpact) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              "controllerStatusCheck",
		DisplayName:     "Controller Status Check",
		Description:     "Checks the BACnet controller is reachable and a sufficient proportion of its devices are responding",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

// controllerHealth aggregates per-device health states for a single BACnet controller (IP address).
// When the proportion of failing devices reaches the threshold, the controller's FaultCheck is marked unhealthy.
type controllerHealth struct {
	faultCheck *healthpb.FaultCheck
	threshold  int // [0,100]: % proportion of failing devices that triggers unhealthy

	mu     sync.Mutex
	states map[string]bool // device SC name → true if currently failing
}

func newControllerHealth(fc *healthpb.FaultCheck, threshold int) *controllerHealth {
	return &controllerHealth{
		faultCheck: fc,
		threshold:  threshold,
		states:     make(map[string]bool),
	}
}

// register adds a device to controller tracking. Idempotent — safe to call on each retry attempt.
func (c *controllerHealth) register(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.states[name]; !exists {
		c.states[name] = false
	}
}

// setFailing marks the named device as failing and recalculates controller health.
func (c *controllerHealth) setFailing(ctx context.Context, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.states[name] = true
	c.recalculate(ctx)
}

// setOK marks the named device as healthy and recalculates controller health.
func (c *controllerHealth) setOK(ctx context.Context, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.states[name] = false
	c.recalculate(ctx)
}

// recalculate must be called with c.mu held.
func (c *controllerHealth) recalculate(ctx context.Context) {
	total := len(c.states)
	if total == 0 {
		return
	}
	failing := 0
	for _, isFailing := range c.states {
		if isFailing {
			failing++
		}
	}
	ratio := failing * 100 / total
	if ratio >= c.threshold {
		c.faultCheck.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
			State: healthpb.HealthCheck_Reliability_CONN_TRANSIENT_FAILURE,
			LastError: &healthpb.HealthCheck_Error{
				SummaryText: fmt.Sprintf("%d/%d devices on controller are failing", failing, total),
				Code:        statusToHealthCode(ControllerUnhealthy),
			},
		})
	} else {
		c.faultCheck.RemoveFault(&healthpb.HealthCheck_Error{
			Code: statusToHealthCode(ControllerUnhealthy),
		})
	}
}
