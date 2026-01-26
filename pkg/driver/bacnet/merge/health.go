package merge

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
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
	ActiveHigh *bool `json:"activeHigh,omitempty"`
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
		if check.ActiveHigh == nil {
			check.ActiveHigh = new(bool)
			*check.ActiveHigh = defaultActiveHigh
		}
	}
	return
}

type Health struct {
	client     *gobacnet.Client
	known      known.Context
	checks     *gen_healthpb.Checks
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	// map of config.HealthCheck.ID to HealthCheck
	DeviceChecks map[string]*gen_healthpb.FaultCheck
	config       healthConfig
	PollTask     *task.Intermittent
}

func NewHealth(client *gobacnet.Client, known known.Context, checks *gen_healthpb.Checks, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*Health, error) {
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
		DeviceChecks: make(map[string]*gen_healthpb.FaultCheck),
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
			h.logger.Warn("skipping Health check with duplicate ID", zap.String("name", bmsPointName), zap.String("checkDisplayName", cc.DisplayName))
			continue
		}

		check, err := h.checks.NewFaultCheck(fmt.Sprintf("%s.%s", h.config.Name, cc.DisplayName), &healthpb.HealthCheck{
			Id:              cc.Id,
			DisplayName:     cc.DisplayName,
			Description:     cc.Description,
			OccupantImpact:  healthpb.HealthCheck_OccupantImpact(cc.OccupantImpact),
			EquipmentImpact: healthpb.HealthCheck_EquipmentImpact(cc.EquipmentImpact),
		})
		if err != nil {
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

	readProcessor := func(source *config.ValueSource, activeHigh bool, id, pointName, errorCode string) {
		readValues = append(readValues, *source)
		requestNames = append(requestNames, pointName)
		resProcessors = append(resProcessors, func(response any) error {
			measured, err := comm.BoolValue(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: pointName, Cause: err}
			}

			if measured == activeHigh {
				if check, ok := h.DeviceChecks[id]; ok {
					raisePointAlarm(pointName, errorCode, "Alarm Detected", check)
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

	for _, checkCfg := range h.config.Checks {
		readProcessor(checkCfg.Source, *checkCfg.ActiveHigh, checkCfg.Id, checkCfg.DisplayName, checkCfg.ErrorCode)
	}

	responses := comm.ReadProperties(ctx, h.client, h.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(ctx, h.faultCheck, h.config.Name, gen_healthpb.TraitName, errs)
	if len(errs) > 0 {
		return fmt.Errorf("health poll errors: %w", multierr.Combine(errs...))
	}
	return nil
}
