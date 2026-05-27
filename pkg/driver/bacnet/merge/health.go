package merge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/bclient"
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
	// If true, and measured value is true, the alarm is active.
	// If false, and measured value is false, the alarm is active.
	ActiveHigh *bool  `json:"activeHigh,omitempty"`
	Summary    string `json:"summary,omitempty"`
	// OKValue switches the check to error code mode. The BACnet point value is read as an
	// integer and compared to OKValue. If they differ, a fault is raised with the actual
	// read value surfaced in the description. Takes precedence over ActiveHigh when set.
	OKValue *int64 `json:"okValue,omitempty"`
}

const defaultActiveHigh = true

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
		if check.OKValue == nil && check.ActiveHigh == nil {
			check.ActiveHigh = new(bool)
			*check.ActiveHigh = defaultActiveHigh
		}
	}
	return
}

type Health struct {
	client     bclient.Client
	known      known.Context
	checks     *healthpb.Checks
	faultCheck *healthpb.FaultCheck
	logger     *zap.Logger

	// map of config.HealthCheck.ID to HealthCheck
	DeviceChecks map[string]*healthpb.FaultCheck
	config       healthConfig
	PollTask     *task.Intermittent
}

func NewHealth(client bclient.Client, known known.Context, checks *healthpb.Checks, faultCheck *healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*Health, error) {
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

	readProcessor := func(source *config.ValueSource, activeHigh bool, id, pointName, errorCode, summary string) {
		readValues = append(readValues, *source)
		requestNames = append(requestNames, pointName)
		resProcessors = append(resProcessors, func(response any) error {
			measured, err := comm.BoolValue(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: pointName, Cause: err}
			}

			if measured == activeHigh {
				if check, ok := h.DeviceChecks[id]; ok {
					raisePointAlarm(pointName, errorCode, summary, fmt.Sprintf("%v", response), check)
				} else {
					h.logger.Warn("no fault check configured for " + pointName)
				}
			} else {
				if check, ok := h.DeviceChecks[id]; ok {
					removePointAlarm(errorCode, check)
				}
			}
			return nil
		})
	}

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
				raisePointAlarm(pointName, errorCode, summary, fmt.Sprintf("%d", val), check)
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
		if checkCfg.OKValue != nil {
			readErrorCodeProcessor(checkCfg.Source, *checkCfg.OKValue, checkCfg.Id, checkCfg.DisplayName, checkCfg.ErrorCode, summary)
		} else {
			readProcessor(checkCfg.Source, *checkCfg.ActiveHigh, checkCfg.Id, checkCfg.DisplayName, checkCfg.ErrorCode, summary)
		}
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
