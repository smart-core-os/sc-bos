package opcua

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

const (
	DriverConfigError = "DriverConfig"
	ServerUnreachable = "ServerUnreachable"

	DeviceConfigError = "DeviceConfig"

	SystemName = "OPCUA"

	// floatTolerance is tolerance to account for floating-point precision issues
	floatTolerance = 1e-9
)

func getDeviceHealthCheck(occupant healthpb.HealthCheck_OccupantImpact, equipment healthpb.HealthCheck_EquipmentImpact) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              "deviceStatusCheck",
		DisplayName:     "Device Status Check",
		Description:     "Checks the device is reachable and responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

func getDeviceErrorCheck(c config.HealthCheck) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              c.Id,
		DisplayName:     c.DisplayName,
		Description:     c.Description,
		OccupantImpact:  healthpb.HealthCheck_OccupantImpact(c.OccupantImpact),
		EquipmentImpact: healthpb.HealthCheck_EquipmentImpact(c.EquipmentImpact),
	}
}

func statusToHealthCode(code string) *healthpb.HealthCheck_Error_Code {
	return &healthpb.HealthCheck_Error_Code{
		Code:   code,
		System: SystemName,
	}
}

func raiseConfigFault(details string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&healthpb.HealthCheck_Error{
		SummaryText: "An issue has been detected with the device's configuration",
		DetailsText: details,
		Code:        statusToHealthCode(DeviceConfigError),
	})
}

// severityMask isolates the severity bits (31:30) of an OPC UA StatusCode.
// See OPC UA Part 4 s7.34: the low bits carry sub-codes and info bits that
// do not affect whether the value is usable. A value delivered through a
// subscription commonly arrives as Good with the Overflow info bit set (0x480),
// which is still a perfectly good value.
const severityMask ua.StatusCode = 0xC0000000

// statusIsGood reports whether the status says the value is usable as-is.
func statusIsGood(c ua.StatusCode) bool { return c&severityMask == ua.StatusGood }

// statusIsUncertain reports whether the status says the value is usable but of reduced quality.
func statusIsUncertain(c ua.StatusCode) bool { return c&severityMask == ua.StatusUncertain }

// statusIsBad reports whether the status says the value should not be used.
func statusIsBad(c ua.StatusCode) bool { return c&severityMask == ua.StatusBad }

// statusHealthCode renders a status code the same way the driver logs it, so the code
// shown in the UI can be matched against the logs.
func statusHealthCode(status ua.StatusCode) *healthpb.HealthCheck_Error_Code {
	return statusToHealthCode(fmt.Sprintf("0x%X", uint32(status)))
}

func setPointReadNotOk(ctx context.Context, nodeId string, status ua.StatusCode, fc *healthpb.FaultCheck) {
	fc.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
		State: healthpb.HealthCheck_Reliability_BAD_RESPONSE,
		LastError: &healthpb.HealthCheck_Error{
			SummaryText: fmt.Sprintf("Attempt to read device point returned non OK status: %s", status.Error()),
			DetailsText: fmt.Sprintf("NodeID: %s, Status: %s", nodeId, status.Error()),
			Code:        statusHealthCode(status),
		},
	})
}

// setPointReadUncertain reports a point whose value is usable but of reduced quality.
// healthpb has no dedicated degraded state, so UNRELIABLE is the closest fit: the value is
// consumed, but our confidence in it is lower than RELIABLE would imply.
func setPointReadUncertain(ctx context.Context, nodeId string, status ua.StatusCode, fc *healthpb.FaultCheck) {
	fc.UpdateReliability(ctx, &healthpb.HealthCheck_Reliability{
		State: healthpb.HealthCheck_Reliability_UNRELIABLE,
		LastError: &healthpb.HealthCheck_Error{
			SummaryText: fmt.Sprintf("Device point returned an uncertain status: %s", status.Error()),
			DetailsText: fmt.Sprintf("NodeID: %s, Status: %s", nodeId, status.Error()),
			Code:        statusHealthCode(status),
		},
	})
}

type Health struct {
	cfg    config.HealthConfig
	logger *zap.Logger

	errorChecks map[string]*healthpb.FaultCheck
	nodeChecks  map[string][]*config.HealthCheck // NodeID -> health checks for that node
}

func readHealthConfig(raw []byte) (cfg config.HealthConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

func newHealth(c config.RawTrait, logger *zap.Logger) (*Health, error) {
	cfg, err := readHealthConfig(c.Raw)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid health config: %w", err)
	}

	nodeChecks := make(map[string][]*config.HealthCheck)
	for i := range cfg.Checks {
		nodeId := cfg.Checks[i].NodeId
		nodeChecks[nodeId] = append(nodeChecks[nodeId], &cfg.Checks[i])
	}

	return &Health{
		cfg:         cfg,
		logger:      logger,
		errorChecks: make(map[string]*healthpb.FaultCheck),
		nodeChecks:  nodeChecks,
	}, nil
}

func raisePointError(point string, code string, summary string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&healthpb.HealthCheck_Error{
		SummaryText: summary,
		DetailsText: "An error has been detected on point: " + point,
		Code:        statusToHealthCode(code),
	})
}

// floatEqual compares two float64 values for equality. Has tolerance in case of floating-point issues.
func floatEqual(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < floatTolerance
}

func (h *Health) handleEvent(_ context.Context, node *ua.NodeID, value any) {
	checks, ok := h.nodeChecks[node.String()]
	if !ok {
		return
	}
	numValue, err := conv.Float64Value(value)
	if err != nil {
		h.logger.Warn("unable to convert value to numeric type for health check",
			zap.String("nodeId", node.String()),
			zap.Any("value", value),
			zap.Error(err))
		return
	}

	for _, hc := range checks {
		if !floatEqual(numValue, *hc.NormalValue) {
			if check, ok := h.errorChecks[hc.Id]; ok {
				raisePointError(hc.Name, hc.ErrorCode, hc.Summary, check)
			} else {
				h.logger.Warn("no fault check found for ID", zap.String("healthCheckId", hc.Id))
			}
		} else {
			if check, ok := h.errorChecks[hc.Id]; ok {
				check.RemoveFault(&healthpb.HealthCheck_Error{
					Code: statusToHealthCode(hc.ErrorCode),
				})
			}
		}
	}
}
