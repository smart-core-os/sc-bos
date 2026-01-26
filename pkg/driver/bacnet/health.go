package bacnet

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

const (
	DriverConfigError = "DriverConfig"
	DeviceUnreachable = "DeviceUnreachable"

	SystemName = "BACnet"
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
		Id:              "traitStatusCheck",
		DisplayName:     fmt.Sprintf("%s Trait Status Check", t.String()),
		Description:     fmt.Sprintf("Checks %s is working correctly", t.String()),
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func updateRequestErrorStatus(ctx context.Context, deviceHealth *gen_healthpb.FaultCheck, name, request string, err error) {
	problemName := fmt.Sprintf("%s.%s", name, "requested")
	if errors.Is(err, context.DeadlineExceeded) {
		deviceHealth.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
			State:          healthpb.HealthCheck_Reliability_CONN_TRANSIENT_FAILURE,
			UnreliableTime: timestamppb.Now(),
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
			State:          healthpb.HealthCheck_Reliability_BAD_RESPONSE,
			UnreliableTime: timestamppb.Now(),
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
		System: SystemName,
	}
}
