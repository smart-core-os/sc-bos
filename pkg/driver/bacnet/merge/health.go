package merge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

type healthConfig struct {
	config.Trait
	// Checks is a map keyed by the BMS alarm name to CheckConfig.
	// It is a map instead of a slice to ensure unique BMS point names.
	Checks map[string]*CheckConfig `json:"checks,omitempty"`
}

type CheckConfig struct {
	config.HealthCheck
	Source *config.ValueSource `json:"source,omitempty"`
	// Deprecated: use OKValue instead. Kept for JSON backwards compatibility;
	// converted to OKValue=0 (activeHigh: true) or OKValue=1 (activeHigh: false) during config parsing if OKValue is not set.
	ActiveHigh *bool  `json:"activeHigh,omitempty"`
	Summary    string `json:"summary,omitempty"`
	// OKValue is the expected integer value of the BACnet point. If the read value differs,
	// a fault is raised with the actual value surfaced in the description.
	OKValue *int64 `json:"okValue,omitempty"`
}

func readHealthConfig(raw []byte) (cfg healthConfig, err error) {
	err = json.Unmarshal(raw, &cfg)

	for name, check := range cfg.Checks {
		if check.Id == "" {
			return cfg, fmt.Errorf("health check %q is missing required field 'id'", name)
		}
		if check.ErrorCode == "" {
			return cfg, fmt.Errorf("health check %q is missing required field 'errorCode'", name)
		}
		if check.Source == nil {
			return cfg, fmt.Errorf("health check %q is missing required field 'source'", name)
		}
		if check.OKValue == nil {
			okVal := int64(0)
			if check.ActiveHigh != nil && !*check.ActiveHigh {
				// activeHigh: false means fault when point reads low (0), so OK value is 1.
				okVal = 1
			}
			check.OKValue = &okVal
		}
	}
	return
}

type Health struct {
	client     *gobacnet.Client
	known      known.Context
	checks     *healthpb.Checks
	faultCheck *healthpb.FaultCheck
	logger     *zap.Logger

	// map of config.HealthCheck.ID to HealthCheck
	DeviceChecks map[string]*healthpb.FaultCheck
	config       healthConfig
	PollTask     *task.Intermittent
}

func NewHealth(client *gobacnet.Client, known known.Context, checks *healthpb.Checks, faultCheck *healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*Health, error) {
	cfg, err := readHealthConfig(config.Raw)
	if err != nil {
		return nil, err
	}

	h := &Health{
		client:     client,
		known:      known,
		checks:     checks,
		faultCheck: faultCheck,
		logger:     logger,

		config:       cfg,
		DeviceChecks: make(map[string]*healthpb.FaultCheck),
	}

	if err := h.initializeChecks(); err != nil {
		return nil, err
	}

	h.PollTask = task.NewIntermittent(h.StartPoll)
	return h, nil
}

func (h *Health) initializeChecks() error {
	// build checks map
	for bmsPointName, cc := range h.config.Checks {
		if h.DeviceChecks[cc.Id] != nil {
			h.logger.Warn("skipping Health check with duplicate ID", zap.String("name", bmsPointName), zap.String("display name", cc.DisplayName))
			continue
		}

		check, err := h.checks.NewFaultCheck(h.config.Name, &healthpb.HealthCheck{
			Id:              cc.Id,
			DisplayName:     cc.DisplayName,
			Description:     cc.Description,
			OccupantImpact:  healthpb.HealthCheck_OccupantImpact(cc.OccupantImpact),
			EquipmentImpact: healthpb.HealthCheck_EquipmentImpact(cc.EquipmentImpact),
		})
		if errors.Is(err, healthpb.ErrAlreadyExists) {
			// A device check with the same (name, id) already exists — the device check already covers this point.
			h.logger.Warn("health check already registered, skipping", zap.String("bmsPoint", bmsPointName), zap.String("id", cc.Id))
			continue
		}
		if err != nil {
			// Dispose any checks we already created to avoid orphaning them.
			for _, created := range h.DeviceChecks {
				created.Dispose()
			}
			h.DeviceChecks = make(map[string]*healthpb.FaultCheck)
			return fmt.Errorf("failed to create fault check for %s: %w", cc.DisplayName, err)
		}

		h.DeviceChecks[cc.Id] = check
	}

	return nil
}

func (h *Health) StartPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "Health", h.config.PollPeriodDuration(), h.config.PollTimeoutDuration(), h.logger, func(ctx context.Context) error {
		err := h.pollPeer(ctx)
		return err
	})
}

func (h *Health) pollPeer(ctx context.Context) error {
	var resProcessors []func(response any) error
	var readValues []config.ValueSource
	var requestNames []string

	readErrorCodeProcessor := func(source *config.ValueSource, okValue int64, id, pointName, errorCode, summary string) {
		readValues = append(readValues, *source)
		requestNames = append(requestNames, pointName)
		resProcessors = append(resProcessors, func(response any) error {
			val, err := comm.IntValue(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: pointName, Cause: err}
			}
			check, ok := h.DeviceChecks[id]
			if !ok {
				h.logger.Warn("no fault check configured for " + pointName)
				return nil
			}
			if val != okValue {
				raisePointAlarm(pointName, errorCode, summary, fmt.Sprintf("%d (expected: %d)", val, okValue), check)
			} else {
				removePointAlarm(errorCode, check)
			}
			return nil
		})
	}

	for _, checkCfg := range h.config.Checks {
		summary := checkCfg.Summary
		if summary == "" {
			summary = "Error Detected"
		}
		readErrorCodeProcessor(checkCfg.Source, *checkCfg.OKValue, checkCfg.Id, checkCfg.DisplayName, checkCfg.ErrorCode, summary)
	}

	responses := comm.ReadProperties(ctx, h.client, h.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(ctx, h.faultCheck, h.config.Name, healthpb.TraitName, errs)
	if len(errs) > 0 {
		return fmt.Errorf("health poll errors: %w", multierr.Combine(errs...))
	}
	return nil
}
