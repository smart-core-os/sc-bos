package bacnet

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/merge"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

const (
	DriverConfigError   = "DriverConfig"
	DeviceUnreachable   = "DeviceUnreachable"
	ControllerUnreachable = "ControllerUnreachable"
)

func createSystemHealthCheck(occupant healthpb.HealthCheck_OccupantImpact, equipment healthpb.HealthCheck_EquipmentImpact) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              "systemStatusCheck",
		DisplayName:     "System Status Check",
		Description:     "Checks the bacnet controller is reachable and the configured nodes are responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

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

func createControllerHealthCheck(ip netip.AddrPort, occupant healthpb.HealthCheck_OccupantImpact, equipment healthpb.HealthCheck_EquipmentImpact) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              "controller/" + ip.String(),
		DisplayName:     "Controller " + ip.Addr().String(),
		Description:     "Checks BACnet controller reachability and device connection ratio",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

// controllerCheck tracks health of a single BACnet IP controller across all its devices.
type controllerCheck struct {
	check  *healthpb.FaultCheck
	mu     sync.Mutex
	states []bool // one slot per device on this controller; true = connected
}

func newControllerCheck(check *healthpb.FaultCheck, deviceCount int) *controllerCheck {
	return &controllerCheck{
		check:  check,
		states: make([]bool, deviceCount),
	}
}

// updateDevice records the latest connected state for device at idx and refreshes the FaultCheck.
func (cc *controllerCheck) updateDevice(ctx context.Context, idx int, connected bool) {
	cc.mu.Lock()
	cc.states[idx] = connected
	total := len(cc.states)
	connCount := 0
	for _, s := range cc.states {
		if s {
			connCount++
		}
	}
	cc.mu.Unlock()

	switch {
	case connCount == total:
		cc.check.ClearFaults()
	case connCount > 0:
		cc.check.SetFault(&healthpb.HealthCheck_Error{
			SummaryText: fmt.Sprintf("%d/%d devices connected", connCount, total),
			Code:        statusToHealthCode(ControllerUnreachable),
		})
	default:
		cc.check.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
			State: healthpb.HealthCheck_Reliability_UNRELIABLE,
			LastError: &healthpb.HealthCheck_Error{
				SummaryText: "No devices connected",
				Code:        statusToHealthCode(ControllerUnreachable),
			},
		})
	}
}

func statusToHealthCode(code string) *healthpb.HealthCheck_Error_Code {
	return &healthpb.HealthCheck_Error_Code{
		Code:   code,
		System: merge.SystemName,
	}
}
