package opcua

import (
	"fmt"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
)

const (
	DriverConfigError = "DriverConfig"
	ServerUnreachable = "ServerUnreachable"

	DeviceConfigError = "DeviceConfig"
)

// this is a generic name that MUST be overridden in concrete device implementations
// to identify separate opcua systems on the same BOS installation
var healthSystemName = "opcua_driver"

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
		System: healthSystemName,
	}
}

func raiseConfigFault(details string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&gen.HealthCheck_Error{
		SummaryText: "An issue has been detected with the device's configuration",
		DetailsText: details,
		Code:        statusToHealthCode(DeviceConfigError),
	})
}

func raisePointFault(nodeId string, errorString string, fc *healthpb.FaultCheck) {

	fc.AddOrUpdateFault(&gen.HealthCheck_Error{
		SummaryText: fmt.Sprintf("Attempt to read device point returned non OK status: %s", errorString),
		DetailsText: fmt.Sprintf("NodeID: %s, Status: %s", nodeId, errorString),
		Code:        getPointHealthCode(nodeId),
	})
}

func clearPointFault(nodeId string, fc *healthpb.FaultCheck) {

	fc.RemoveFault(&gen.HealthCheck_Error{
		Code: getPointHealthCode(nodeId),
	})
}

func getPointHealthCode(nodeId string) *gen.HealthCheck_Error_Code {
	// each nodeId within the opcua system will be unique, and we can only have 1 fault per node at a time,
	// so the nodeId alone is sufficient to identify the fault programmatically. The details will provide more context.
	return &gen.HealthCheck_Error_Code{
		Code:   nodeId,
		System: healthSystemName,
	}
}
