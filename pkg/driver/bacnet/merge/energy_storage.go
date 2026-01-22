package merge

import (
	"context"
	"encoding/json"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/energystoragepb"
)

type energyStorageConfig struct {
	config.Trait
	EnergyKwh  *config.ValueSource `json:"energyKwh,omitempty"`
	Percentage *config.ValueSource `json:"percentage,omitempty"`
}

func readEnergyStorageConfig(raw []byte) (cfg energyStorageConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

type energyStorage struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	model *energystoragepb.Model
	*energystoragepb.ModelServer
	config   energyStorageConfig
	pollTask *task.Intermittent
}

func newEnergyStorage(client *gobacnet.Client, devices known.Context, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*energyStorage, error) {
	cfg, err := readEnergyStorageConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := energystoragepb.NewModel(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.1),
	)))
	e := &energyStorage{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: energystoragepb.NewModelServer(model),
		config:      cfg,
	}
	e.pollTask = task.NewIntermittent(e.startPoll)
	return e, nil
}

func (e *energyStorage) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "energy storage", e.config.PollPeriodDuration(), e.config.PollTimeoutDuration(), e.logger, func(ctx context.Context) error {
		_, err := e.pollPeer(ctx)
		return err
	})
}

func (e *energyStorage) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(e.config.Name, node.HasTrait(trait.EnergyStorage, node.WithClients(energystoragepb.WrapApi(e))))
}

func (e *energyStorage) GetEnergyLevel(ctx context.Context, request *traits.GetEnergyLevelRequest) (*traits.EnergyLevel, error) {
	_, err := e.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return e.ModelServer.GetEnergyLevel(ctx, request)
}

func (e *energyStorage) PullEnergyLevel(request *traits.PullEnergyLevelRequest, server grpc.ServerStreamingServer[traits.PullEnergyLevelResponse]) error {
	_ = e.pollTask.Attach(server.Context())
	return e.ModelServer.PullEnergyLevel(request, server)
}

func (e *energyStorage) pollPeer(ctx context.Context) (*traits.EnergyLevel, error) {
	data := &traits.EnergyLevel{}
	var resProcessors []func(response any) error
	var readValues []config.ValueSource
	var requestNames []string

	if e.config.EnergyKwh != nil {
		requestNames = append(requestNames, "energyKwh")
		readValues = append(readValues, *e.config.EnergyKwh)
		resProcessors = append(resProcessors, func(response any) error {
			energyKwh, err := comm.Float32Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "energyKwh", Cause: err}
			}
			if data.Quantity == nil {
				data.Quantity = &traits.EnergyLevel_Quantity{}
			}
			data.Quantity.EnergyKwh = energyKwh
			return nil
		})
	}
	if e.config.Percentage != nil {
		requestNames = append(requestNames, "percentage")
		readValues = append(readValues, *e.config.Percentage)
		resProcessors = append(resProcessors, func(response any) error {
			percentage, err := comm.Float32Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "percentage", Cause: err}
			}
			if data.Quantity == nil {
				data.Quantity = &traits.EnergyLevel_Quantity{}
			}
			data.Quantity.Percentage = percentage
			return nil
		})
	}
	responses := comm.ReadProperties(ctx, e.client, e.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}

	updateTraitFaultCheck(e.faultCheck, e.config.Name, trait.EnergyStorage, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}

	return e.model.UpdateEnergyLevel(data)
}
