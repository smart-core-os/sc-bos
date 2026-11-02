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
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
)

type meterConfig struct {
	config.Trait
	Production *config.ValueSource `json:"production,omitempty"`
	Usage      *config.ValueSource `json:"usage,omitempty"`
	Unit       string              `json:"unit,omitempty"`
}

func readMeterConfig(raw []byte) (cfg meterConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

type meterTrait struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *healthpb.FaultCheck
	logger     *zap.Logger

	model *meterpb.Model
	*meterpb.ModelServer
	config   meterConfig
	pollTask *task.Intermittent
}

func newMeter(client *gobacnet.Client, devices known.Context, faultCheck *healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*meterTrait, error) {
	cfg, err := readMeterConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := meterpb.NewModel(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.0001),
	)))
	t := &meterTrait{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: meterpb.NewModelServer(model),
		config:      cfg,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	return t, nil
}

// meterReadingSupport describes what this meter reports, based on which value sources are configured.
// Production and usage share the configured unit, a meter measures import and export in the same commodity.
// The produced unit is left empty unless production is read: consumers, dbo.MeterFields among them, use its
// presence to decide whether the meter reports export at all, and a spurious unit yields a constant-zero series.
func meterReadingSupport(cfg meterConfig) *meterpb.MeterReadingSupport {
	support := &meterpb.MeterReadingSupport{
		ResourceSupport: &typespb.ResourceSupport{Readable: true, Observable: true},
		UsageUnit:       cfg.Unit,
	}
	if cfg.Production != nil {
		support.ProducedUnit = cfg.Unit
	}
	return support
}

func (t *meterTrait) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(t.config.Name,
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(t)),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{
			MeterReading: meterReadingSupport(t.config),
		})),
		node.HasTrait(meterpb.TraitName),
	)
}

func (t *meterTrait) GetMeterReading(ctx context.Context, request *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	_, err := t.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return t.ModelServer.GetMeterReading(ctx, request)
}

func (t *meterTrait) PullMeterReadings(request *meterpb.PullMeterReadingsRequest, server meterpb.MeterApi_PullMeterReadingsServer) error {
	err := t.pollTask.Attach(server.Context())
	if err != nil {
		return err
	}

	// avoid returning the zero value if we are the first to attach since reboot
	timeoutCtx, cleanup := context.WithTimeout(server.Context(), t.config.PollTimeoutDuration())
	defer cleanup()
	for change := range t.model.PullMeterReadings(timeoutCtx) {
		if change.Value.Usage != 0 || change.Value.Produced != 0 {
			break
		}
	}

	return t.ModelServer.PullMeterReadings(request, server)
}

func (t *meterTrait) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "meter", t.config.PollPeriodDuration(), t.config.PollTimeoutDuration(), t.logger, func(ctx context.Context) error {
		_, err := t.pollPeer(ctx)
		return err
	})
}

func (t *meterTrait) pollPeer(ctx context.Context) (*meterpb.MeterReading, error) {
	data := &meterpb.MeterReading{}
	var readValues []config.ValueSource
	var resProcessors []func(response any) error

	if t.config.Usage != nil {
		readValues = append(readValues, *t.config.Usage)
		resProcessors = append(resProcessors, func(response any) error {
			usage, err := comm.Float32Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "usage", Cause: err}
			}
			data.Usage = usage
			return nil
		})
	}
	if t.config.Production != nil {
		readValues = append(readValues, *t.config.Production)
		resProcessors = append(resProcessors, func(response any) error {
			produced, err := comm.Float32Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "production", Cause: err}
			}
			data.Produced = produced
			return nil
		})
	}

	responses := comm.ReadProperties(ctx, t.client, t.known, readValues...)
	var errs []error
	for i, response := range responses {
		if err := resProcessors[i](response); err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(ctx, t.faultCheck, t.config.Name, trait.Meter, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	return t.model.UpdateMeterReading(data)
}

func (t *meterTrait) PollTask() *task.Intermittent { return t.pollTask }
