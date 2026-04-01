package merge

import (
	"context"
	"encoding/json"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
)

type temperatureConfig struct {
	config.Trait

	Measured *config.ValueSource `json:"measured,omitempty"`
	SetPoint *config.ValueSource `json:"setPoint,omitempty"`
}

func readTemperatureConfig(raw []byte) (cfg temperatureConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

type temperature struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *healthpb.FaultCheck
	logger     *zap.Logger

	model *temperaturepb.Model
	*temperaturepb.ModelServer
	config   temperatureConfig
	pollTask *task.Intermittent
}

func newTemperature(client *gobacnet.Client, devices known.Context, faultCheck *healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*temperature, error) {
	cfg, err := readTemperatureConfig(config.Raw)
	if err != nil {
		return nil, err
	}

	model := temperaturepb.NewModel(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.1), // report temperature changes of 0.1C or more
	)))
	t := &temperature{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: temperaturepb.NewModelServer(model),
		config:      cfg,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	return t, nil
}

func (t *temperature) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "temperature", t.config.PollPeriodDuration(), t.config.PollTimeoutDuration(), t.logger, func(ctx context.Context) error {
		_, err := t.pollPeer(ctx)
		return err
	})
}

func (t *temperature) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(t.config.Name,
		node.HasServer(temperaturepb.RegisterTemperatureApiServer, temperaturepb.TemperatureApiServer(t)),
		node.HasTrait(temperaturepb.TraitName),
	)
}

func (t *temperature) GetTemperature(ctx context.Context, request *temperaturepb.GetTemperatureRequest) (*temperaturepb.Temperature, error) {
	_, err := t.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return t.ModelServer.GetTemperature(ctx, request)
}

func (t *temperature) UpdateTemperature(ctx context.Context, request *temperaturepb.UpdateTemperatureRequest) (*temperaturepb.Temperature, error) {
	if request.GetTemperature() == nil || request.GetTemperature().SetPoint == nil {
		return t.GetTemperature(ctx, &temperaturepb.GetTemperatureRequest{Name: request.Name})
	}
	newSetPoint := float32(request.GetTemperature().SetPoint.ValueCelsius)

	if t.config.SetPoint != nil {
		err := comm.WriteProperty(ctx, t.client, t.known, *t.config.SetPoint, newSetPoint, 0)
		if err != nil {
			return nil, err
		}
	}

	return pollUntil(ctx, t.config.DefaultRWConsistencyTimeoutDuration(), t.pollPeer, func(temperature *temperaturepb.Temperature) bool {
		return temperature.SetPoint.ValueCelsius == float64(newSetPoint)
	})
}

func (t *temperature) PullTemperature(request *temperaturepb.PullTemperatureRequest, server temperaturepb.TemperatureApi_PullTemperatureServer) error {
	_ = t.pollTask.Attach(server.Context())
	return t.ModelServer.PullTemperature(request, server)
}

func (t *temperature) pollPeer(ctx context.Context) (*temperaturepb.Temperature, error) {
	data := &temperaturepb.Temperature{}
	var resProcessors []func(response any) error
	var readValues []config.ValueSource

	if t.config.Measured != nil {
		readValues = append(readValues, *t.config.Measured)
		resProcessors = append(resProcessors, func(response any) error {
			measured, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "measured", Cause: err}
			}
			data.Measured = &typespb.Temperature{ValueCelsius: measured}
			return nil
		})
	}

	if t.config.SetPoint != nil {
		readValues = append(readValues, *t.config.SetPoint)
		resProcessors = append(resProcessors, func(response any) error {
			setPoint, err := comm.Float64Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "setPoint", Cause: err}
			}
			data.SetPoint = &typespb.Temperature{ValueCelsius: setPoint}
			return nil
		})
	}
	responses := comm.ReadProperties(ctx, t.client, t.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(ctx, t.faultCheck, t.config.Name, temperaturepb.TraitName, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	return t.model.UpdateTemperature(data)
}
