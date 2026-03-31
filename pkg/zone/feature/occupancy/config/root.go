package config

import (
	"encoding/json"

	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	OccupancySensors []string `json:"occupancySensors,omitempty"`
	// These EnterLeave devices get converted to occupancy using `EnterTotal - LeaveTotal`
	EnterLeaveOccupancySensors   []string                      `json:"enterLeaveOccupancySensors,omitempty"`
	EnterLeaveOccupancySensorSLA *EnterLeaveOccupancySensorSLA `json:"enterLeaveOccupancySensorSLA,omitempty"`
}

type EnterLeaveOccupancySensorSLA struct {
	// CantFail is a list of sensor names to define whether Pull can't return an error without causing overall zone failure
	CantFail []string `json:"cantFail,omitempty"`
	// PercentageOfAcceptableFailures is the percentage of sensors in the zone that can fail without causing overall zone failure.
	// Acceptable values are 0-100. For example, if set to 25, up to 25% of sensors can fail without causing overall zone failure,
	// Unless at least one of those failures is in the CantFail list.
	// If omitted or set to 0, any failure will cause overall zone failure.
	// If set to 100, all sensors can fail without causing overall zone failure unless one of the sensors is in the CantFail list.
	// The default is 0 if it is set to an invalid value.
	PercentageOfAcceptableFailures float64 `json:"percentageOfAcceptableFailures,omitempty"`
}

func ParseConfig(data []byte) (Root, error) {
	var cfg Root
	err := json.Unmarshal(data, &cfg)

	if err != nil {
		return cfg, err
	}

	if cfg.EnterLeaveOccupancySensorSLA != nil {
		if cfg.EnterLeaveOccupancySensorSLA.PercentageOfAcceptableFailures < 0 || cfg.EnterLeaveOccupancySensorSLA.PercentageOfAcceptableFailures > 100 {
			cfg.EnterLeaveOccupancySensorSLA.PercentageOfAcceptableFailures = 0
		}
	}

	return cfg, nil
}
