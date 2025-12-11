package hpd

import (
	"github.com/smart-core-os/sc-bos/pkg/gen"
)

const (
	SystemName = "Steinel HPD"

	BadResponse = "BadResponse"
	DriverError = "DriverError"
	Offline     = "Offline"
)

var (
	// This health check monitors the device to check if it is online and communicating properly.
	commsHealthCheck = &gen.HealthCheck{
		Id:              "commsCheck",
		DisplayName:     "Comms Check",
		Description:     "Checks if the device is online and communicating properly",
		OccupantImpact:  gen.HealthCheck_COMFORT,
		EquipmentImpact: gen.HealthCheck_FUNCTION,
	}

	noResponse = &gen.HealthCheck_Reliability{
		State: gen.HealthCheck_Reliability_NO_RESPONSE,
		LastError: &gen.HealthCheck_Error{
			SummaryText: "Device Offline",
			DetailsText: "No communication received from device since the last Smart Core restart",
			Code: &gen.HealthCheck_Error_Code{
				Code:   Offline,
				System: SystemName,
			},
		},
	}

	badResponse = &gen.HealthCheck_Reliability{
		State: gen.HealthCheck_Reliability_BAD_RESPONSE,
		LastError: &gen.HealthCheck_Error{
			SummaryText: "Bad Response",
			DetailsText: "The device has sent an unexpected response to a request",
			Code: &gen.HealthCheck_Error_Code{
				Code:   BadResponse,
				System: SystemName,
			},
		},
	}

	driverError = &gen.HealthCheck_Error{
		SummaryText: "Internal Driver Error",
		DetailsText: "An unexpected error occurred within the driver",
		Code: &gen.HealthCheck_Error_Code{
			Code:   DriverError,
			System: SystemName,
		},
	}
)
