package merge

import (
	"context"
	"encoding/json"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
)

type modeDataPoints struct {
	FanOnValue     *float32
	HeatingOnValue *float32
	CoolingOnValue *float32
}

type airTempModeConfig struct {
	FanOn *config.ValueSource `json:"fanOn,omitempty"`
	// When FanOn reads equal to or above this value, the fan is considered on. Defaults to 1.
	FanOnThreshold *float32            `json:"fanOnThreshold,omitempty"`
	HeatingOn      *config.ValueSource `json:"heatingOn,omitempty"`
	// When HeatingOn reads equal to or above this value, the heating is considered on. Defaults to 1.
	HeatingOnThreshold *float32            `json:"heatingOnThreshold,omitempty"`
	CoolingOn          *config.ValueSource `json:"coolingOn,omitempty"`
	// When CoolingOn reads equal to or above this value, the cooling is considered on. Defaults to 1.
	CoolingOnThreshold *float32 `json:"coolingOnThreshold,omitempty"`
}

type airTemperatureConfig struct {
	config.Trait
	SetPoint           *config.ValueSource `json:"setPoint,omitempty"`
	AmbientTemperature *config.ValueSource `json:"ambientTemperature,omitempty"`
	AmbientHumidity    *config.ValueSource `json:"ambientHumidity,omitempty"`
	SetPointLow        *config.ValueSource `json:"setPointLow,omitempty"`
	SetPointHigh       *config.ValueSource `json:"setPointHigh,omitempty"`
	// SetPointDeadBand should be defined when SetPointLow & SetPointHigh are, defaults to 1
	SetPointDeadBand *float32           `json:"deadBand,omitempty,omitzero"`
	ModeConfig       *airTempModeConfig `json:"modeConfig,omitempty"`
}

func readAirTemperatureConfig(raw []byte) (cfg airTemperatureConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	if err == nil {
		if cfg.SetPointDeadBand == nil || *cfg.SetPointDeadBand == 0 {
			cfg.SetPointDeadBand = new(float32)
			*cfg.SetPointDeadBand = 1
		}
		if cfg.ModeConfig != nil {
			if cfg.ModeConfig.FanOnThreshold == nil {
				cfg.ModeConfig.FanOnThreshold = new(float32)
				*cfg.ModeConfig.FanOnThreshold = 1
			}
			if cfg.ModeConfig.HeatingOnThreshold == nil {
				cfg.ModeConfig.HeatingOnThreshold = new(float32)
				*cfg.ModeConfig.HeatingOnThreshold = 1
			}
			if cfg.ModeConfig.CoolingOnThreshold == nil {
				cfg.ModeConfig.CoolingOnThreshold = new(float32)
				*cfg.ModeConfig.CoolingOnThreshold = 1
			}
		}
	}
	return
}

type airTemperature struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	model *airtemperaturepb.Model
	*airtemperaturepb.ModelServer
	config   airTemperatureConfig
	pollTask *task.Intermittent
}

func newAirTemperature(client *gobacnet.Client, devices known.Context, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*airTemperature, error) {
	cfg, err := readAirTemperatureConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := airtemperaturepb.NewModel(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.1), // report temperature changes of 0.1C or more
	)))
	t := &airTemperature{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: airtemperaturepb.NewModelServer(model),
		config:      cfg,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	return t, nil
}

func (t *airTemperature) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "air temperature", t.config.PollPeriodDuration(), t.config.PollTimeoutDuration(), t.logger, func(ctx context.Context) error {
		_, err := t.pollPeer(ctx)
		return err
	})
}

func (t *airTemperature) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(t.config.Name, node.HasTrait(trait.AirTemperature, node.WithClients(airtemperaturepb.WrapApi(t))))
}

func (t *airTemperature) GetAirTemperature(ctx context.Context, request *traits.GetAirTemperatureRequest) (*traits.AirTemperature, error) {
	_, err := t.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return t.ModelServer.GetAirTemperature(ctx, request)
}

func (t *airTemperature) UpdateAirTemperature(ctx context.Context, request *traits.UpdateAirTemperatureRequest) (*traits.AirTemperature, error) {
	if request.GetState().GetTemperatureSetPoint() == nil {
		return t.GetAirTemperature(ctx, &traits.GetAirTemperatureRequest{Name: request.Name})
	}
	newSetPoint := float32(request.GetState().GetTemperatureSetPoint().GetValueCelsius())

	if t.config.SetPointLow != nil && t.config.SetPointHigh != nil {
		deadBand := *t.config.SetPointDeadBand
		err := comm.WriteProperty(ctx, t.client, t.known, *t.config.SetPointLow, newSetPoint-deadBand, 0)
		if err != nil {
			t.logger.Error("WriteProperty SetPointLow", zap.Error(err))
			return nil, err
		}

		err = comm.WriteProperty(ctx, t.client, t.known, *t.config.SetPointHigh, newSetPoint+deadBand, 0)
		if err != nil {
			t.logger.Error("WriteProperty SetPointHigh", zap.Error(err))
			return nil, err
		}
	}

	if t.config.SetPoint != nil {
		err := comm.WriteProperty(ctx, t.client, t.known, *t.config.SetPoint, newSetPoint, 0)
		if err != nil {
			return nil, err
		}
	}

	// todo: not strictly correct as we're not paying attention to the require customisation properties that ModelServer would give us
	return pollUntil(ctx, t.config.DefaultRWConsistencyTimeoutDuration(), t.pollPeer, func(temperature *traits.AirTemperature) bool {
		return temperature.GetTemperatureSetPoint().ValueCelsius == float64(newSetPoint)
	})
}

func (t *airTemperature) PullAirTemperature(request *traits.PullAirTemperatureRequest, server traits.AirTemperatureApi_PullAirTemperatureServer) error {
	_ = t.pollTask.Attach(server.Context())
	return t.ModelServer.PullAirTemperature(request, server)
}

// pollPeer fetches data from the peer device and saves the data locally.
func (t *airTemperature) pollPeer(ctx context.Context) (*traits.AirTemperature, error) {
	data := &traits.AirTemperature{}
	var resProcessors []func(response any) error
	var readValues []config.ValueSource
	var requestNames []string
	modeData := &modeDataPoints{}

	if t.config.SetPoint != nil {
		requestNames = append(requestNames, "setPoint")
		readValues = append(readValues, *t.config.SetPoint)
		resProcessors = append(resProcessors, func(response any) error {
			setPoint, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "setPoint", Cause: err}
			}
			data.TemperatureGoal = &traits.AirTemperature_TemperatureSetPoint{
				TemperatureSetPoint: &types.Temperature{ValueCelsius: setPoint},
			}
			return nil
		})
	}

	if t.config.SetPointHigh != nil {
		requestNames = append(requestNames, "setPointHigh")
		readValues = append(readValues, *t.config.SetPointHigh)
		resProcessors = append(resProcessors, func(response any) error {
			setPointHigh, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "setPointHigh", Cause: err}
			}
			setPoint := setPointHigh - float64(*t.config.SetPointDeadBand)
			data.TemperatureGoal = &traits.AirTemperature_TemperatureSetPoint{
				TemperatureSetPoint: &types.Temperature{ValueCelsius: setPoint},
			}
			return nil
		})
	} else if t.config.SetPointLow != nil {
		requestNames = append(requestNames, "setPointLow")
		readValues = append(readValues, *t.config.SetPointLow)
		resProcessors = append(resProcessors, func(response any) error {
			setPointLow, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "setPointLow", Cause: err}
			}
			setPoint := setPointLow + float64(*t.config.SetPointDeadBand)
			data.TemperatureGoal = &traits.AirTemperature_TemperatureSetPoint{
				TemperatureSetPoint: &types.Temperature{ValueCelsius: setPoint},
			}
			return nil
		})
	}

	if t.config.AmbientTemperature != nil {
		requestNames = append(requestNames, "ambientTemperature")
		readValues = append(readValues, *t.config.AmbientTemperature)
		resProcessors = append(resProcessors, func(response any) error {
			ambientTemperature, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "ambientTemperature", Cause: err}
			}
			data.AmbientTemperature = &types.Temperature{ValueCelsius: ambientTemperature}
			return nil
		})
	}

	if t.config.AmbientHumidity != nil {
		requestNames = append(requestNames, "ambientHumidity")
		readValues = append(readValues, *t.config.AmbientHumidity)
		resProcessors = append(resProcessors, func(response any) error {
			ambientHumidity, err := comm.Float32Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "ambientHumidity", Cause: err}
			}
			data.AmbientHumidity = &ambientHumidity
			return nil
		})
	}

	if t.config.ModeConfig != nil {
		modeData = t.getModePoints(&resProcessors, &readValues, &requestNames)
	}
	responses := comm.ReadProperties(ctx, t.client, t.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(t.faultCheck, t.config.Name, trait.AirTemperature, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	if t.config.ModeConfig != nil {
		updateMode(t.config.ModeConfig, modeData, data)
	}
	return t.model.UpdateAirTemperature(data)
}

// getModePoints appends all the data points needed to calculate the current AirTemperature.Mode to the provided slices.
func (t *airTemperature) getModePoints(processors *[]func(response any) error, values *[]config.ValueSource, names *[]string) *modeDataPoints {
	data := &modeDataPoints{}
	if t.config.ModeConfig.FanOn != nil {
		*names = append(*names, "fanOn")
		*values = append(*values, *t.config.ModeConfig.FanOn)
		*processors = append(*processors, func(response any) error {
			fanOn, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "fanOn", Cause: err}
			}
			data.FanOnValue = new(float32)
			*data.FanOnValue = float32(fanOn)
			return nil
		})
	}

	if t.config.ModeConfig.HeatingOn != nil {
		*names = append(*names, "heatingOn")
		*values = append(*values, *t.config.ModeConfig.HeatingOn)
		*processors = append(*processors, func(response any) error {
			heatingOn, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "heatingOn", Cause: err}
			}
			data.HeatingOnValue = new(float32)
			*data.HeatingOnValue = float32(heatingOn)
			return nil
		})
	}

	if t.config.ModeConfig.CoolingOn != nil {
		*names = append(*names, "coolingOn")
		*values = append(*values, *t.config.ModeConfig.CoolingOn)
		*processors = append(*processors, func(response any) error {
			coolingOn, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "coolingOn", Cause: err}
			}
			data.CoolingOnValue = new(float32)
			*data.CoolingOnValue = float32(coolingOn)
			return nil
		})
	}
	return data
}

// updateMode updates the AirTemperature.Mode based on the current values of the mode data points.
func updateMode(modeCfg *airTempModeConfig, data *modeDataPoints, airTemp *traits.AirTemperature) {

	const (
		OFF  = -1
		NONE = 0
		ON   = 1
	)
	calcState := func(v, t *float32) int {
		switch {
		case v == nil:
			return NONE
		case *v >= *t:
			return ON
		default:
			return OFF
		}
	}

	fan := calcState(data.FanOnValue, modeCfg.FanOnThreshold)
	heat := calcState(data.HeatingOnValue, modeCfg.HeatingOnThreshold)
	cool := calcState(data.CoolingOnValue, modeCfg.CoolingOnThreshold)

	m := traits.AirTemperature_MODE_UNSPECIFIED
	switch {
	case fan == OFF:
		m = traits.AirTemperature_OFF
	case heat == ON && cool == ON:
		m = traits.AirTemperature_HEAT_COOL
	case heat == ON:
		m = traits.AirTemperature_HEAT
	case cool == ON:
		m = traits.AirTemperature_COOL
	case fan == ON:
		m = traits.AirTemperature_FAN_ONLY
	}

	airTemp.Mode = m
}
