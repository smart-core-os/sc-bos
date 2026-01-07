package pestsense

import (
	"encoding/json"
	"errors"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
)

type Response struct {
	Source                     string `json:"source"`
	Detections                 int    `json:"detections"`
	PacketType                 int    `json:"packettype"`
	IndividualDeviceDetections int    `json:"individualdevicedetections"`
	DeviceNumber               string `json:"devicenumber"`
	DeviceId                   int    `json:"deviceid"`
	Action                     string `json:"action"`
}

func handleResponse(body []byte, devices map[string]*pestSensor, logger *zap.Logger) {

	response := Response{}

	err := json.Unmarshal(body, &response)

	if err != nil {
		logger.Error("failed to unmarshal MQTT response", zap.Error(err), zap.ByteString("payload", body))
		return
	}

	logger.Debug("processing device", zap.String("deviceId", response.DeviceNumber))
	occupied, err := getOccupied(response.PacketType)
	if err != nil {
		logger.Warn("unexpected packet type", zap.Int("packetType", response.PacketType))
		return
	}
	logger.Debug("device occupancy state", zap.Bool("occupied", occupied))

	device, exists := devices[response.DeviceNumber]

	if exists {
		if occupied {
			logger.Debug("setting device state", zap.String("deviceId", response.DeviceNumber), zap.Bool("occupied", true))
			_, _ = device.occupancy.Set(&traits.Occupancy{State: traits.Occupancy_OCCUPIED, PeopleCount: int32(response.IndividualDeviceDetections)})
		} else {
			logger.Debug("setting device state", zap.String("deviceId", response.DeviceNumber), zap.Bool("occupied", false))
			_, _ = device.occupancy.Set(&traits.Occupancy{State: traits.Occupancy_UNOCCUPIED, PeopleCount: int32(response.IndividualDeviceDetections)})
		}
	} else {
		logger.Warn("received data for unknown device", zap.String("deviceNumber", response.DeviceNumber))
	}
}

func getOccupied(packetType int) (bool, error) {
	switch packetType {
	case 4:
		return true, nil
	case 6:
		return false, nil
	default:
		return false, errors.New("unexpected packet type")
	}
}
