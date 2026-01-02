package opcua

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gopcua/opcua/ua"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
)

const (
	DriverConfigError = "DriverConfig"
	ServerUnreachable = "ServerUnreachable"

	DeviceConfigError = "DeviceConfig"

	SystemName = "OPCUA"
)

func getSystemHealthCheck(occupant gen.HealthCheck_OccupantImpact, equipment gen.HealthCheck_EquipmentImpact) *gen.HealthCheck {
	return &gen.HealthCheck{
		Id:              "systemStatusCheck",
		DisplayName:     "System Status Check",
		Description:     "Checks the opc ua server is reachable and the configured nodes are responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func getDeviceHealthCheck(occupant gen.HealthCheck_OccupantImpact, equipment gen.HealthCheck_EquipmentImpact) *gen.HealthCheck {
	return &gen.HealthCheck{
		Id:              "deviceStatusCheck",
		DisplayName:     "Device Status Check",
		Description:     "Checks the device is reachable and responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func statusToHealthCode(code string) *gen.HealthCheck_Error_Code {
	return &gen.HealthCheck_Error_Code{
		Code:   code,
		System: SystemName,
	}
}

func raiseConfigFault(details string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&gen.HealthCheck_Error{
		SummaryText: "An issue has been detected with the device's configuration",
		DetailsText: details,
		Code:        statusToHealthCode(DeviceConfigError),
	})
}

func setPointReadNotOk(ctx context.Context, nodeId string, status ua.StatusCode, fc *healthpb.FaultCheck) {
	fc.UpdateReliability(ctx, &gen.HealthCheck_Reliability{
		State: gen.HealthCheck_Reliability_BAD_RESPONSE,
		LastError: &gen.HealthCheck_Error{
			SummaryText: fmt.Sprintf("Attempt to read device point returned non OK status: %s", status.Error()),
			DetailsText: fmt.Sprintf("NodeID: %s, Status: %s", nodeId, status.Error()),
			Code:        statusToHealthCode(strconv.Itoa(int(status))),
		},
	})
}
