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

	FireAlarm             CheckConfig `json:"fireAlarm,omitempty"`
	CommRmDxNo1Fault      CheckConfig `json:"commRmDxNo1Fault,omitempty"`
	CommRmDxNo2Fault      CheckConfig `json:"commRmDxNo2Fault,omitempty"`
	DBCommRmDxNo1Fault    CheckConfig `json:"dbCommRmDxNo1Fault,omitempty"`
	DBCommRmDxNo2Fault    CheckConfig `json:"dbCommRmDxNo2Fault,omitempty"`
	MCWBoosterFault       CheckConfig `json:"mcwBoosterFault,omitempty"`
	MCWTankNo1LowLevel    CheckConfig `json:"mcwTankNo1LowLevel,omitempty"`
	MCWTankNo1HighLevel   CheckConfig `json:"mcwTankNo1HighLevel,omitempty"`
	MCWTankNo2LowLevel    CheckConfig `json:"mcwTankNo2LowLevel,omitempty"`
	MCWTankNo2HighLevel   CheckConfig `json:"mcwTankNo2HighLevel,omitempty"`
	HWSSecondaryPumpFault CheckConfig `json:"hwsSecondaryPumpFault,omitempty"`
	SprinklerSystemFault  CheckConfig `json:"sprinklerSystemFault,omitempty"`
	CommRmDxNo3Fault      CheckConfig `json:"commRmDxNo3Fault,omitempty"`
	CommRmDxNo4Fault      CheckConfig `json:"commRmDxNo4Fault,omitempty"`
	DBCommRmDxNo3Fault    CheckConfig `json:"dbCommRmDxNo3Fault,omitempty"`
	DBCommRmDxNo4Fault    CheckConfig `json:"dbCommRmDxNo4Fault,omitempty"`
	OATFrostActive        CheckConfig `json:"oatFrostActive,omitempty"`
	ASHPFrostActive       CheckConfig `json:"ashpFrostActive,omitempty"`
	HWSLowTemp            CheckConfig `json:"hwsLowTemp,omitempty"`
	AHU4SupplyFanFault    CheckConfig `json:"ahu4SupplyFanFault,omitempty"`
	AHU4ExtractFanFault   CheckConfig `json:"ahu4ExtractFanFault,omitempty"`
	EHThermCutout         CheckConfig `json:"ehThermCutout,omitempty"`
	GeneratorFault        CheckConfig `json:"generatorFault,omitempty"`
	SmokeVentFault        CheckConfig `json:"smokeVentFault,omitempty"`
	AHU1SupplyFan1Fault   CheckConfig `json:"ahu1SupplyFan1Fault,omitempty"`
	AHU1SupplyFan2Fault   CheckConfig `json:"ahu1SupplyFan2Fault,omitempty"`
	AHU1SupplyFan3Fault   CheckConfig `json:"ahu1SupplyFan3Fault,omitempty"`
	AHU1ExtractFan1Fault  CheckConfig `json:"ahu1ExtractFan1Fault,omitempty"`
	AHU1ExtractFan2Fault  CheckConfig `json:"ahu1ExtractFan2Fault,omitempty"`
	AHU1ExtractFan3Fault  CheckConfig `json:"ahu1ExtractFan3Fault,omitempty"`
	AHU1STG1CMP1Fault     CheckConfig `json:"ahu1STG1CMP1Fault,omitempty"`
	AHU1STG1CMP2Fault     CheckConfig `json:"ahu1STG1CMP2Fault,omitempty"`
	AHU1STG1CMP3Fault     CheckConfig `json:"ahu1STG1CMP3Fault,omitempty"`
	AHU1STG1CMP4Fault     CheckConfig `json:"ahu1STG1CMP4Fault,omitempty"`
	AHU2SupplyFan1Fault   CheckConfig `json:"ahu2SupplyFan1Fault,omitempty"`
	AHU2SupplyFan2Fault   CheckConfig `json:"ahu2SupplyFan2Fault,omitempty"`
	AHU2ExtractFan1Fault  CheckConfig `json:"ahu2ExtractFan1Fault,omitempty"`
	AHU2ExtractFan2Fault  CheckConfig `json:"ahu2ExtractFan2Fault,omitempty"`
	AHU2STG1CMP1Fault     CheckConfig `json:"ahu2STG1CMP1Fault,omitempty"`
	AHU2STG1CMP2Fault     CheckConfig `json:"ahu2STG1CMP2Fault,omitempty"`
	AHU2STG1CMP3Fault     CheckConfig `json:"ahu2STG1CMP3Fault,omitempty"`
	AHU2STG1CMP4Fault     CheckConfig `json:"ahu2STG1CMP4Fault,omitempty"`
	AHU3SupplyFanFault    CheckConfig `json:"ahu3SupplyFanFault,omitempty"`
	AHU3ExtractFanFault   CheckConfig `json:"ahu3ExtractFanFault,omitempty"`
	AHU3CMP1Fault         CheckConfig `json:"ahu3CMP1Fault,omitempty"`
	AHU3CMP2Fault         CheckConfig `json:"ahu3CMP2Fault,omitempty"`
}

type CheckConfig struct {
	config.HealthCheck
	Source *config.ValueSource `json:"source,omitempty"`
}

func readHealthConfig(raw []byte) (cfg healthConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
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
	checkConfigs := []CheckConfig{
		h.config.AHU1ExtractFan1Fault,
		h.config.AHU1ExtractFan2Fault,
		h.config.AHU1ExtractFan3Fault,
		h.config.AHU1SupplyFan1Fault,
		h.config.AHU1SupplyFan2Fault,
		h.config.AHU1SupplyFan3Fault,
		h.config.AHU1STG1CMP1Fault,
		h.config.AHU1STG1CMP2Fault,
		h.config.AHU1STG1CMP3Fault,
		h.config.AHU1STG1CMP4Fault,
		h.config.AHU2ExtractFan1Fault,
		h.config.AHU2ExtractFan2Fault,
		h.config.AHU2SupplyFan1Fault,
		h.config.AHU2SupplyFan2Fault,
		h.config.AHU2STG1CMP1Fault,
		h.config.AHU2STG1CMP2Fault,
		h.config.AHU2STG1CMP3Fault,
		h.config.AHU2STG1CMP4Fault,
		h.config.AHU3CMP1Fault,
		h.config.AHU3CMP2Fault,
		h.config.AHU3ExtractFanFault,
		h.config.AHU3SupplyFanFault,
		h.config.AHU4ExtractFanFault,
		h.config.AHU4SupplyFanFault,
		h.config.ASHPFrostActive,
		h.config.CommRmDxNo1Fault,
		h.config.CommRmDxNo2Fault,
		h.config.CommRmDxNo3Fault,
		h.config.CommRmDxNo4Fault,
		h.config.DBCommRmDxNo1Fault,
		h.config.DBCommRmDxNo2Fault,
		h.config.DBCommRmDxNo3Fault,
		h.config.DBCommRmDxNo4Fault,
		h.config.EHThermCutout,
		h.config.FireAlarm,
		h.config.GeneratorFault,
		h.config.HWSLowTemp,
		h.config.HWSSecondaryPumpFault,
		h.config.MCWBoosterFault,
		h.config.MCWTankNo1HighLevel,
		h.config.MCWTankNo1LowLevel,
		h.config.MCWTankNo2HighLevel,
		h.config.MCWTankNo2LowLevel,
		h.config.OATFrostActive,
		h.config.SmokeVentFault,
		h.config.SprinklerSystemFault,
	}
	for _, cc := range checkConfigs {
		if cc.Id == "" || h.DeviceChecks[cc.Id] != nil {
			h.logger.Warn("Skipping Health check with missing or duplicate ID", zap.String("name", h.config.Name), zap.String("checkDisplayName", cc.DisplayName))
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

	setup := func(source *config.ValueSource, id, pointName, errorCode string) {
		readValues = append(readValues, *source)
		requestNames = append(requestNames, pointName)
		resProcessors = append(resProcessors, func(response any) error {
			measured, err := comm.BoolValue(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: pointName, Cause: err}
			}

			if measured {
				if check, ok := h.DeviceChecks[id]; ok {
					raisePointAlarm(pointName, errorCode, "Alarm Detected", check)
				} else {
					h.logger.Warn("No fault check configured for " + pointName)
				}
			} else {
				if check, ok := h.DeviceChecks[id]; ok {
					removePointAlarm(errorCode, check)
				}
			}
			return nil
		})
	}

	if h.config.FireAlarm.Source != nil {
		setup(h.config.FireAlarm.Source, h.config.FireAlarm.Id, "Fire Alarm", h.config.FireAlarm.ErrorCode)
	}

	if h.config.CommRmDxNo1Fault.Source != nil {
		setup(h.config.CommRmDxNo1Fault.Source, h.config.CommRmDxNo1Fault.Id, "Comm RM DX No 1 Fault", h.config.CommRmDxNo1Fault.ErrorCode)
	}

	if h.config.CommRmDxNo2Fault.Source != nil {
		setup(h.config.CommRmDxNo2Fault.Source, h.config.CommRmDxNo2Fault.Id, "Comm RM DX No 2 Fault", h.config.CommRmDxNo2Fault.ErrorCode)
	}

	if h.config.DBCommRmDxNo1Fault.Source != nil {
		setup(h.config.DBCommRmDxNo1Fault.Source, h.config.DBCommRmDxNo1Fault.Id, "DB Comm RM DX No 1 Fault", h.config.DBCommRmDxNo1Fault.ErrorCode)
	}

	if h.config.DBCommRmDxNo2Fault.Source != nil {
		setup(h.config.DBCommRmDxNo2Fault.Source, h.config.DBCommRmDxNo2Fault.Id, "DB Comm RM DX No 2 Fault", h.config.DBCommRmDxNo2Fault.ErrorCode)
	}

	if h.config.MCWBoosterFault.Source != nil {
		setup(h.config.MCWBoosterFault.Source, h.config.MCWBoosterFault.Id, "MCW Booster Fault", h.config.MCWBoosterFault.ErrorCode)
	}

	if h.config.MCWTankNo1LowLevel.Source != nil {
		setup(h.config.MCWTankNo1LowLevel.Source, h.config.MCWTankNo1LowLevel.Id, "MCW Tank No 1 Low Level", h.config.MCWTankNo1LowLevel.ErrorCode)
	}

	if h.config.MCWTankNo1HighLevel.Source != nil {
		setup(h.config.MCWTankNo1HighLevel.Source, h.config.MCWTankNo1HighLevel.Id, "MCW Tank No 1 High Level", h.config.MCWTankNo1HighLevel.ErrorCode)
	}

	if h.config.MCWTankNo2LowLevel.Source != nil {
		setup(h.config.MCWTankNo2LowLevel.Source, h.config.MCWTankNo2LowLevel.Id, "MCW Tank No 2 Low Level", h.config.MCWTankNo2LowLevel.ErrorCode)
	}

	if h.config.MCWTankNo2HighLevel.Source != nil {
		setup(h.config.MCWTankNo2HighLevel.Source, h.config.MCWTankNo2HighLevel.Id, "MCW Tank No 2 High Level", h.config.MCWTankNo2HighLevel.ErrorCode)
	}

	if h.config.HWSSecondaryPumpFault.Source != nil {
		setup(h.config.HWSSecondaryPumpFault.Source, h.config.HWSSecondaryPumpFault.Id, "HWS Secondary Pump Fault", h.config.HWSSecondaryPumpFault.ErrorCode)
	}

	if h.config.SprinklerSystemFault.Source != nil {
		setup(h.config.SprinklerSystemFault.Source, h.config.SprinklerSystemFault.Id, "Sprinkler System Fault", h.config.SprinklerSystemFault.ErrorCode)
	}

	if h.config.CommRmDxNo3Fault.Source != nil {
		setup(h.config.CommRmDxNo3Fault.Source, h.config.CommRmDxNo3Fault.Id, "Comm RM DX No 3 Fault", h.config.CommRmDxNo3Fault.ErrorCode)
	}

	if h.config.CommRmDxNo4Fault.Source != nil {
		setup(h.config.CommRmDxNo4Fault.Source, h.config.CommRmDxNo4Fault.Id, "Comm RM DX No 4 Fault", h.config.CommRmDxNo4Fault.ErrorCode)
	}

	if h.config.DBCommRmDxNo3Fault.Source != nil {
		setup(h.config.DBCommRmDxNo3Fault.Source, h.config.DBCommRmDxNo3Fault.Id, "DB Comm RM DX No 3 Fault", h.config.DBCommRmDxNo3Fault.ErrorCode)
	}

	if h.config.DBCommRmDxNo4Fault.Source != nil {
		setup(h.config.DBCommRmDxNo4Fault.Source, h.config.DBCommRmDxNo4Fault.Id, "DB Comm RM DX No 4 Fault", h.config.DBCommRmDxNo4Fault.ErrorCode)
	}

	if h.config.OATFrostActive.Source != nil {
		setup(h.config.OATFrostActive.Source, h.config.OATFrostActive.Id, "OAT Frost Active", h.config.OATFrostActive.ErrorCode)
	}

	if h.config.ASHPFrostActive.Source != nil {
		setup(h.config.ASHPFrostActive.Source, h.config.ASHPFrostActive.Id, "ASHP Frost Active", h.config.ASHPFrostActive.ErrorCode)
	}

	if h.config.HWSLowTemp.Source != nil {
		setup(h.config.HWSLowTemp.Source, h.config.HWSLowTemp.Id, "HWS Low Temp", h.config.HWSLowTemp.ErrorCode)
	}

	if h.config.AHU4SupplyFanFault.Source != nil {
		setup(h.config.AHU4SupplyFanFault.Source, h.config.AHU4SupplyFanFault.Id, "AHU4 Supply Fan Fault", h.config.AHU4SupplyFanFault.ErrorCode)
	}

	if h.config.AHU4ExtractFanFault.Source != nil {
		setup(h.config.AHU4ExtractFanFault.Source, h.config.AHU4ExtractFanFault.Id, "AHU4 Extract Fan Fault", h.config.AHU4ExtractFanFault.ErrorCode)
	}

	if h.config.EHThermCutout.Source != nil {
		setup(h.config.EHThermCutout.Source, h.config.EHThermCutout.Id, "EH Therm Cutout", h.config.EHThermCutout.ErrorCode)
	}

	if h.config.GeneratorFault.Source != nil {
		setup(h.config.GeneratorFault.Source, h.config.GeneratorFault.Id, "Generator Fault", h.config.GeneratorFault.ErrorCode)
	}

	if h.config.SmokeVentFault.Source != nil {
		setup(h.config.SmokeVentFault.Source, h.config.SmokeVentFault.Id, "Smoke Vent Fault", h.config.SmokeVentFault.ErrorCode)
	}

	if h.config.AHU1ExtractFan1Fault.Source != nil {
		setup(h.config.AHU1ExtractFan1Fault.Source, h.config.AHU1ExtractFan1Fault.Id, "AHU1 Extract Fan 1 Fault", h.config.AHU1ExtractFan1Fault.ErrorCode)
	}

	if h.config.AHU1ExtractFan2Fault.Source != nil {
		setup(h.config.AHU1ExtractFan2Fault.Source, h.config.AHU1ExtractFan2Fault.Id, "AHU1 Extract Fan 2 Fault", h.config.AHU1ExtractFan2Fault.ErrorCode)
	}

	if h.config.AHU1ExtractFan3Fault.Source != nil {
		setup(h.config.AHU1ExtractFan3Fault.Source, h.config.AHU1ExtractFan3Fault.Id, "AHU1 Extract Fan 3 Fault", h.config.AHU1ExtractFan3Fault.ErrorCode)
	}

	if h.config.AHU1SupplyFan1Fault.Source != nil {
		setup(h.config.AHU1SupplyFan1Fault.Source, h.config.AHU1SupplyFan1Fault.Id, "AHU1 Supply Fan 1 Fault", h.config.AHU1SupplyFan1Fault.ErrorCode)
	}

	if h.config.AHU1SupplyFan2Fault.Source != nil {
		setup(h.config.AHU1SupplyFan2Fault.Source, h.config.AHU1SupplyFan2Fault.Id, "AHU1 Supply Fan 2 Fault", h.config.AHU1SupplyFan2Fault.ErrorCode)
	}

	if h.config.AHU1SupplyFan3Fault.Source != nil {
		setup(h.config.AHU1SupplyFan3Fault.Source, h.config.AHU1SupplyFan3Fault.Id, "AHU1 Supply Fan 3 Fault", h.config.AHU1SupplyFan3Fault.ErrorCode)
	}

	if h.config.AHU2ExtractFan1Fault.Source != nil {
		setup(h.config.AHU2ExtractFan1Fault.Source, h.config.AHU2ExtractFan1Fault.Id, "AHU2 Extract Fan 1 Fault", h.config.AHU2ExtractFan1Fault.ErrorCode)
	}

	if h.config.AHU2ExtractFan2Fault.Source != nil {
		setup(h.config.AHU2ExtractFan2Fault.Source, h.config.AHU2ExtractFan2Fault.Id, "AHU2 Extract Fan 2 Fault", h.config.AHU2ExtractFan2Fault.ErrorCode)
	}

	if h.config.AHU2SupplyFan1Fault.Source != nil {
		setup(h.config.AHU2SupplyFan1Fault.Source, h.config.AHU2SupplyFan1Fault.Id, "AHU2 Supply Fan 1 Fault", h.config.AHU2SupplyFan1Fault.ErrorCode)
	}

	if h.config.AHU2SupplyFan2Fault.Source != nil {
		setup(h.config.AHU2SupplyFan2Fault.Source, h.config.AHU2SupplyFan2Fault.Id, "AHU2 Supply Fan 2 Fault", h.config.AHU2SupplyFan2Fault.ErrorCode)
	}

	if h.config.AHU3ExtractFanFault.Source != nil {
		setup(h.config.AHU3ExtractFanFault.Source, h.config.AHU3ExtractFanFault.Id, "AHU3 Extract Fan Fault", h.config.AHU3ExtractFanFault.ErrorCode)
	}
	if h.config.AHU3SupplyFanFault.Source != nil {
		setup(h.config.AHU3SupplyFanFault.Source, h.config.AHU3SupplyFanFault.Id, "AHU3 Supply Fan Fault", h.config.AHU3SupplyFanFault.ErrorCode)
	}

	if h.config.AHU3CMP1Fault.Source != nil {
		setup(h.config.AHU3CMP1Fault.Source, h.config.AHU3CMP1Fault.Id, "AHU3 CMP 1 Fault", h.config.AHU3CMP1Fault.ErrorCode)
	}

	if h.config.AHU3CMP2Fault.Source != nil {
		setup(h.config.AHU3CMP2Fault.Source, h.config.AHU3CMP2Fault.Id, "AHU3 CMP 2 Fault", h.config.AHU3CMP2Fault.ErrorCode)
	}

	if h.config.AHU2STG1CMP1Fault.Source != nil {
		setup(h.config.AHU2STG1CMP1Fault.Source, h.config.AHU2STG1CMP1Fault.Id, "AHU2 STG 1 CMP 1 Fault", h.config.AHU2STG1CMP1Fault.ErrorCode)
	}

	if h.config.AHU2STG1CMP2Fault.Source != nil {
		setup(h.config.AHU2STG1CMP2Fault.Source, h.config.AHU2STG1CMP2Fault.Id, "AHU2 STG 1 CMP 2 Fault", h.config.AHU2STG1CMP2Fault.ErrorCode)
	}

	if h.config.AHU2STG1CMP3Fault.Source != nil {
		setup(h.config.AHU2STG1CMP3Fault.Source, h.config.AHU2STG1CMP3Fault.Id, "AHU2 STG 1 CMP 3 Fault", h.config.AHU2STG1CMP3Fault.ErrorCode)
	}

	if h.config.AHU2STG1CMP4Fault.Source != nil {
		setup(h.config.AHU2STG1CMP4Fault.Source, h.config.AHU2STG1CMP4Fault.Id, "AHU2 STG 1 CMP 4 Fault", h.config.AHU2STG1CMP4Fault.ErrorCode)
	}

	if h.config.AHU1STG1CMP1Fault.Source != nil {
		setup(h.config.AHU1STG1CMP1Fault.Source, h.config.AHU1STG1CMP1Fault.Id, "AHU1 STG 1 CMP 1 Fault", h.config.AHU1STG1CMP1Fault.ErrorCode)
	}

	if h.config.AHU1STG1CMP2Fault.Source != nil {
		setup(h.config.AHU1STG1CMP2Fault.Source, h.config.AHU1STG1CMP2Fault.Id, "AHU1 STG 1 CMP 2 Fault", h.config.AHU1STG1CMP2Fault.ErrorCode)
	}

	if h.config.AHU1STG1CMP3Fault.Source != nil {
		setup(h.config.AHU1STG1CMP3Fault.Source, h.config.AHU1STG1CMP3Fault.Id, "AHU1 STG 1 CMP 3 Fault", h.config.AHU1STG1CMP3Fault.ErrorCode)
	}

	if h.config.AHU1STG1CMP4Fault.Source != nil {
		setup(h.config.AHU1STG1CMP4Fault.Source, h.config.AHU1STG1CMP4Fault.Id, "AHU1 STG 1 CMP 4 Fault", h.config.AHU1STG1CMP4Fault.ErrorCode)
	}

	responses := comm.ReadProperties(ctx, h.client, h.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(h.faultCheck, h.config.Name, gen_healthpb.TraitName, errs)
	if len(errs) > 0 {
		return fmt.Errorf("health poll errors: %w", multierr.Combine(errs...))
	}
	return nil
}
