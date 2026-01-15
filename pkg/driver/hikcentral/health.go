package hikcentral

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/api"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

const (
	SystemName = "HikCentral"

	BadResponse = "BadResponse"
	Offline     = "Offline"
)

var (
	// This health check monitors the device to check if it is online and communicating properly.
	// Also checks whether the device is self-reporting any faults/alarms.
	deviceHealthCheck = &gen_healthpb.HealthCheck{
		Id:          "deviceCheck",
		DisplayName: "Device Check",
		Description: "Checks if the device is online, communicating properly and if there are any device alarms",
		// not sure about the impact, if the CCTV is not functioning, it is safety/security
		OccupantImpact:  gen_healthpb.HealthCheck_HEALTH,
		EquipmentImpact: gen_healthpb.HealthCheck_FUNCTION,
	}
)

func noResponseReliability() *gen_healthpb.HealthCheck_Reliability {
	return &gen_healthpb.HealthCheck_Reliability{
		State: gen_healthpb.HealthCheck_Reliability_NO_RESPONSE,
		LastError: &gen_healthpb.HealthCheck_Error{
			SummaryText: "Device Offline",
			DetailsText: "No communication received from device since the last Smart Core restart",
			Code: &gen_healthpb.HealthCheck_Error_Code{
				Code:   Offline,
				System: SystemName,
			},
		},
	}
}

func badResponseReliability() *gen_healthpb.HealthCheck_Reliability {
	return &gen_healthpb.HealthCheck_Reliability{
		State: gen_healthpb.HealthCheck_Reliability_BAD_RESPONSE,
		LastError: &gen_healthpb.HealthCheck_Error{
			SummaryText: "Bad Response",
			DetailsText: "The device has sent an unexpected response to a request",
			Code: &gen_healthpb.HealthCheck_Error_Code{
				Code:   BadResponse,
				System: SystemName,
			},
		},
	}
}

func updateReliability(ctx context.Context, err error, fc *healthpb.FaultCheck) {
	var rel *gen_healthpb.HealthCheck_Reliability

	if err != nil {
		rel = noResponseReliability()
		var unsupportedTypeErr *json.UnmarshalTypeError
		var badStatusErr *badStatusError
		if errors.As(err, &unsupportedTypeErr) || errors.As(err, &badStatusErr) {
			rel = badResponseReliability()
		}
	} else {
		// When reliable, create a new object with nil LastError
		rel = &gen_healthpb.HealthCheck_Reliability{
			State: gen_healthpb.HealthCheck_Reliability_RELIABLE,
		}
	}

	fc.UpdateReliability(ctx, rel)
}

func updateDeviceFaults(faults allFaults, fc *healthpb.FaultCheck) {

	for alarmType, active := range faults {
		if active && alarmType != api.CameraRecordingRecovered {
			fc.AddOrUpdateFault(&gen_healthpb.HealthCheck_Error{
				SummaryText: getFaultSummary(alarmType),
				DetailsText: getFaultDetails(alarmType),
				Code:        statusToHealthCode(alarmType),
			})
		}
	}

	// Remove any faults in Smart Core that are no longer present
	for _, faultType := range api.AlarmTypes {
		if !faults[faultType] {
			fc.RemoveFault(&gen_healthpb.HealthCheck_Error{
				Code: statusToHealthCode(faultType),
			})
		}
	}
}

func getFaultSummary(alarmType string) string {
	switch alarmType {
	case api.VideoLossAlarm:
		return "Video Loss"
	case api.VideoTamperingAlarm:
		return "Video Tampering"
	case api.CameraRecordingExceptionAlarm:
		return "Camera Recording Exception"
	default:
		return "Unknown Alarm"
	}
}

func getFaultDetails(alarmType string) string {
	switch alarmType {
	case api.VideoLossAlarm:
		return "The camera has lost video signal"
	case api.VideoTamperingAlarm:
		return "Video tampering has been detected"
	case api.CameraRecordingExceptionAlarm:
		return "An exception occurred during camera recording"
	default:
		return "An unknown alarm has been detected"
	}
}

func statusToHealthCode(status string) *gen_healthpb.HealthCheck_Error_Code {
	return &gen_healthpb.HealthCheck_Error_Code{
		Code:   status,
		System: SystemName,
	}
}
