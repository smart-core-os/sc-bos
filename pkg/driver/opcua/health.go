package opcua

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gopcua/opcua/ua"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

const (
	DriverConfigError = "DriverConfig"
	ServerUnreachable = "ServerUnreachable"

	DeviceConfigError = "DeviceConfig"

	SystemName = "OPCUA"
)

func getSystemHealthCheck(occupant gen_healthpb.HealthCheck_OccupantImpact, equipment gen_healthpb.HealthCheck_EquipmentImpact) *gen_healthpb.HealthCheck {
	return &gen_healthpb.HealthCheck{
		Id:              "systemStatusCheck",
		DisplayName:     "System Status Check",
		Description:     "Checks the opc ua server is reachable and the configured nodes are responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func getDeviceHealthCheck(occupant gen_healthpb.HealthCheck_OccupantImpact, equipment gen_healthpb.HealthCheck_EquipmentImpact) *gen_healthpb.HealthCheck {
	return &gen_healthpb.HealthCheck{
		Id:              "deviceStatusCheck",
		DisplayName:     "Device Status Check",
		Description:     "Checks the device is reachable and responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func statusToHealthCode(code string) *gen_healthpb.HealthCheck_Error_Code {
	return &gen_healthpb.HealthCheck_Error_Code{
		Code:   code,
		System: SystemName,
	}
}

func raiseConfigFault(details string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&gen_healthpb.HealthCheck_Error{
		SummaryText: "An issue has been detected with the device's configuration",
		DetailsText: details,
		Code:        statusToHealthCode(DeviceConfigError),
	})
}

func setPointReadNotOk(ctx context.Context, nodeId string, status ua.StatusCode, fc *healthpb.FaultCheck) {
	fc.UpdateReliability(ctx, &gen_healthpb.HealthCheck_Reliability{
		State: gen_healthpb.HealthCheck_Reliability_BAD_RESPONSE,
		LastError: &gen_healthpb.HealthCheck_Error{
			SummaryText: fmt.Sprintf("Attempt to read device point returned non OK status: %s", status.Error()),
			DetailsText: fmt.Sprintf("NodeID: %s, Status: %s", nodeId, status.Error()),
			Code:        statusToHealthCode(strconv.Itoa(int(status))),
		},
	})
}
