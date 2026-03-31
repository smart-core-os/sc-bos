package hpd

import "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"

const (
	SystemName = "Steinel HPD"

	BadResponse = "BadResponse"
	DriverError = "DriverError"
	Offline     = "Offline"
)

var (
	// This health check monitors the device to check if it is online and communicating properly.
	commsHealthCheck = &healthpb.HealthCheck{
		Id:              "commsCheck",
		DisplayName:     "Comms Check",
		Description:     "Checks if the device is online and communicating properly",
		OccupantImpact:  healthpb.HealthCheck_COMFORT,
		EquipmentImpact: healthpb.HealthCheck_FUNCTION,
	}

	noResponse = &healthpb.HealthCheck_Reliability{
		State: healthpb.HealthCheck_Reliability_NO_RESPONSE,
		LastError: &healthpb.HealthCheck_Error{
			SummaryText: "Device Offline",
			DetailsText: "No communication received from device since the last Smart Core restart",
			Code: &healthpb.HealthCheck_Error_Code{
				Code:   Offline,
				System: SystemName,
			},
		},
	}

	badResponse = &healthpb.HealthCheck_Reliability{
		State: healthpb.HealthCheck_Reliability_BAD_RESPONSE,
		LastError: &healthpb.HealthCheck_Error{
			SummaryText: "Bad Response",
			DetailsText: "The device has sent an unexpected response to a request",
			Code: &healthpb.HealthCheck_Error_Code{
				Code:   BadResponse,
				System: SystemName,
			},
		},
	}

	driverError = &healthpb.HealthCheck_Error{
		SummaryText: "Internal Driver Error",
		DetailsText: "An unexpected error occurred within the driver",
		Code: &healthpb.HealthCheck_Error_Code{
			Code:   DriverError,
			System: SystemName,
		},
	}
)
