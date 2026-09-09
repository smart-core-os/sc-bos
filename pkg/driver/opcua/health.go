package opcua

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

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
	// PointSubscribeError reports points the driver could not subscribe to but is still
	// retrying. It is deliberately not DeviceConfigError: a point failing for a reason that
	// may go away is not a misconfigured point, and calling it one sends whoever is reading
	// the fault hunting a mistake in the config that isn't there.
	PointSubscribeError = "PointSubscribe"

	SystemName = "OPCUA"

	// floatTolerance is tolerance to account for floating-point precision issues
	floatTolerance = 1e-9
)

func getDeviceHealthCheck(occupant healthpb.HealthCheck_OccupantImpact, equipment healthpb.HealthCheck_EquipmentImpact) *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              deviceStatusCheckId,
		DisplayName:     "Device Status Check",
		Description:     "Checks the device is reachable and responding correctly",
		OccupantImpact:  occupant,
		EquipmentImpact: equipment,
	}
}

const (
	// deviceStatusCheckId is the check carrying whether the device is doing its job: its
	// trait-backed points, plus the server-level signals.
	deviceStatusCheckId = "deviceStatusCheck"
	// informationalPointCheckId is the check carrying failures on points marked informational.
	//
	// Both are relative ids. The registry namespaces them by owner, so what a client sees over
	// the devices API is healthpb.AbsID(owner, id) - "<driver name>:deviceStatusCheck".
	informationalPointCheckId = "informationalPointCheck"
)

// getInformationalPointCheck builds the check that failures on config.Variable.Informational
// points are reported on, keeping them off deviceStatusCheck.
//
// The impacts are hard-coded rather than taken from the device: by definition no trait reads
// these points, so the device is still doing its job and the equipment is still running. That
// is also what keeps this check low in the Health page's impact breakdown, where it is still
// listed.
func getInformationalPointCheck() *healthpb.HealthCheck {
	return &healthpb.HealthCheck{
		Id:              informationalPointCheckId,
		DisplayName:     "Informational Point Check",
		Description:     "Checks points the device exports but no trait reads",
		OccupantImpact:  healthpb.HealthCheck_NO_OCCUPANT_IMPACT,
		EquipmentImpact: healthpb.HealthCheck_NO_EQUIPMENT_IMPACT,
	}
}

// hasInformationalVariable reports whether the device marks any point informational, and so
// whether it needs the second check at all.
func hasInformationalVariable(dev *config.Device) bool {
	for _, v := range dev.Variables {
		if v.Informational {
			return true
		}
	}
	return false
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

func configFault(details string) *healthpb.HealthCheck_Error {
	return &healthpb.HealthCheck_Error{
		SummaryText: "An issue has been detected with the device's configuration",
		DetailsText: details,
		Code:        statusToHealthCode(DeviceConfigError),
	}
}

func raiseConfigFault(details string, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(configFault(details))
}

func raisePointSubscribeFault(details string, count int, fc *healthpb.FaultCheck) {
	fc.AddOrUpdateFault(&healthpb.HealthCheck_Error{
		SummaryText: fmt.Sprintf("Could not subscribe to %d of the device's points, retrying", count),
		DetailsText: details,
		Code:        statusToHealthCode(PointSubscribeError),
	})
}

// pointHealth aggregates the subscribe state of one device's points into that device's fault
// check.
//
// The aggregation is forced rather than chosen: faults are keyed on system and code, so a
// fault raised per point would have every point overwriting its neighbours. One fault per
// state, naming the affected points in its details, is the only shape that survives.
//
// It carries its own mutex because the goroutines behind it - one pump per device, plus that
// device's straggler retry loop - all write here. That does not make the underlying healthpb
// check goroutine-safe: checkBase.write is an unsynchronised read-modify-write, which this
// driver already races from those same goroutines through handleStatusValue. Batching the
// points of a device onto one subscription narrows that race from one goroutine per point to
// one per device, but does not close it. The mutex only keeps the maps below consistent.
type pointHealth struct {
	fc *healthpb.FaultCheck

	mu sync.Mutex
	// transient holds node id -> last error for points that failed and are still being retried.
	transient map[string]string
	// permanent holds node id -> error for points the server says can never be subscribed.
	permanent map[string]string
}

func newPointHealth(fc *healthpb.FaultCheck) *pointHealth {
	return &pointHealth{
		fc:        fc,
		transient: make(map[string]string),
		permanent: make(map[string]string),
	}
}

// setFailing records a point that failed to subscribe for a reason that may go away.
func (p *pointHealth) setFailing(nodeId string, err error) {
	p.applyBatch(nil, map[string]error{nodeId: err}, nil)
}

// setPermanent records a point we have given up on because the server says it can never work.
func (p *pointHealth) setPermanent(nodeId string, err error) {
	p.applyBatch(nil, nil, map[string]error{nodeId: err})
}

// setOk records a point that is subscribed, clearing whatever was last said about it.
func (p *pointHealth) setOk(nodeId string) {
	p.applyBatch([]string{nodeId}, nil, nil)
}

// applyBatch records the outcome of one subscribe attempt for a whole device at once: ok names
// the points now subscribed, failing those that failed for a reason that may go away, and
// permanent those the server says can never work. A point named in more than one is resolved
// in that order, so the last word wins.
//
// One call rather than one per point because commit rewrites both faults and pushes the check
// to the registry every time. With a subscription per point those calls were spread over as
// many goroutines; with one per device they arrive together, and a 200-point device would
// otherwise publish 200 growing versions of the same fault per attempt, every one of them
// visible over the devices API.
func (p *pointHealth) applyBatch(ok []string, failing, permanent map[string]error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, nodeId := range ok {
		delete(p.transient, nodeId)
		delete(p.permanent, nodeId)
	}
	for nodeId, err := range failing {
		delete(p.permanent, nodeId)
		p.transient[nodeId] = err.Error()
	}
	for nodeId, err := range permanent {
		delete(p.transient, nodeId)
		p.permanent[nodeId] = err.Error()
	}
	p.commit()
}

// commit rewrites both faults from the current state, removing the one whose bucket has
// emptied. Callers must hold p.mu.
//
// Rewriting on every change is cheap because checkBase.write drops an update that would leave
// the check unchanged, which only holds while identical state renders identical text - hence
// the sorted node ids in renderPoints.
//
// Note this makes pointHealth the owner of both fault codes on a device's check: it clears
// DeviceConfigError whenever no point is permanently failing, so anything else raising that
// code on the same check would be wiped out here. Nothing else does today, and a second
// source of device config faults should get its own code rather than share this one.
func (p *pointHealth) commit() {
	if len(p.transient) == 0 {
		p.fc.RemoveFault(&healthpb.HealthCheck_Error{Code: statusToHealthCode(PointSubscribeError)})
	} else {
		raisePointSubscribeFault("Retrying the subscription to:\n"+renderPoints(p.transient), len(p.transient), p.fc)
	}

	if len(p.permanent) == 0 {
		p.fc.RemoveFault(&healthpb.HealthCheck_Error{Code: statusToHealthCode(DeviceConfigError)})
	} else {
		raiseConfigFault("Gave up subscribing to:\n"+renderPoints(p.permanent), p.fc)
	}
}

// renderPoints lists points and their errors one per line, sorted by node id so that an
// unchanged set of failures always renders identically.
func renderPoints(points map[string]string) string {
	var sb strings.Builder
	for _, nodeId := range slices.Sorted(maps.Keys(points)) {
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(nodeId)
		sb.WriteString(": ")
		sb.WriteString(points[nodeId])
	}
	return sb.String()
}

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
