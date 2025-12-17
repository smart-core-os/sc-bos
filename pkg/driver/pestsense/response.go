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

func handleResponse(body []byte, devices map[string]*PestSensor, logger *zap.Logger) {

	response := Response{}

	err := json.Unmarshal(body, &response)

	if err != nil {
		logger.Error("Error unmarshalling")
		return
	}

	logger.Debug("ID: " + response.DeviceNumber)
	occupied, err := getOccupied(response.PacketType)
	if err != nil {
		logger.Warn("Unexpected packet type")
		return
	}
	logger.Debug("Occupied: ", zap.Bool("occupied", occupied))

	device, exists := devices[response.DeviceNumber]

	if exists {
		if occupied {
			logger.Debug("Setting occupied for device " + response.DeviceNumber)
			_, _ = device.occupancy.Set(&traits.Occupancy{State: traits.Occupancy_OCCUPIED, PeopleCount: int32(response.IndividualDeviceDetections)})
		} else {
			logger.Debug("Setting unoccupied for device " + response.DeviceNumber)
			_, _ = device.occupancy.Set(&traits.Occupancy{State: traits.Occupancy_UNOCCUPIED, PeopleCount: int32(response.IndividualDeviceDetections)})
		}
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
