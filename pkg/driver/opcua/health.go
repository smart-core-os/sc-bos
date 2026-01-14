package opcua

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
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

func getDeviceErrorCheck(c config.HealthCheck) *gen_healthpb.HealthCheck {
	return &gen_healthpb.HealthCheck{
		Id:              c.Id,
		DisplayName:     c.DisplayName,
		Description:     c.Description,
		OccupantImpact:  gen_healthpb.HealthCheck_OccupantImpact(c.OccupantImpact),
		EquipmentImpact: gen_healthpb.HealthCheck_EquipmentImpact(c.EquipmentImpact),
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

func raisePointError(point string, code string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&gen_healthpb.HealthCheck_Error{
		SummaryText: "An error has been detected on point: " + point,
		DetailsText: "An error has been detected on point: " + point,
		Code:        statusToHealthCode(code),
	})
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
		if numValue < *hc.OkLowerBound || numValue > *hc.OkUpperBound {
			if check, ok := h.errorChecks[hc.Id]; ok {
				raisePointError(hc.Name, hc.ErrorCode, check)
			} else {
				h.logger.Warn("no fault check found for ID", zap.String("healthCheckId", hc.Id))
			}
		} else {
			if check, ok := h.errorChecks[hc.Id]; ok {
				check.RemoveFault(&gen_healthpb.HealthCheck_Error{
					Code: statusToHealthCode(hc.ErrorCode),
				})
			}
		}
	}
}
