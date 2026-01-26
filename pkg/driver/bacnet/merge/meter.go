package merge

import (
	"context"
	"encoding/json"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

type meterConfig struct {
	config.Trait
	Usage *config.ValueSource `json:"usage,omitempty"`
	Unit  string              `json:"unit,omitempty"`
}

func readMeterConfig(raw []byte) (cfg meterConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

type meterTrait struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	model *meter.Model
	*meter.ModelServer
	config   meterConfig
	pollTask *task.Intermittent
}

func newMeter(client *gobacnet.Client, devices known.Context, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*meterTrait, error) {
	cfg, err := readMeterConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := meter.NewModel(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.0001),
	)))
	t := &meterTrait{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: meter.NewModelServer(model),
		config:      cfg,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	return t, nil
}

func (t *meterTrait) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(t.config.Name, node.HasTrait(meter.TraitName, node.WithClients(meterpb.WrapApi(t), meterpb.WrapInfo(&meter.InfoServer{
		MeterReading: &meterpb.MeterReadingSupport{
			ResourceSupport: &types.ResourceSupport{Readable: true, Observable: true},
			UsageUnit:       t.config.Unit,
		},
	}))))
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
		if change.Value.Usage != 0 {
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
	responses := comm.ReadProperties(ctx, t.client, t.known, *t.config.Usage)
	var errs []error
	usage, err := comm.Float32Value(responses[0])
	if err != nil {
		errs = append(errs, comm.ErrReadProperty{Prop: "usage", Cause: err})
	}
	updateTraitFaultCheck(ctx, t.faultCheck, t.config.Name, trait.Meter, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	data := &meterpb.MeterReading{
		Usage: usage,
	}
	return t.model.UpdateMeterReading(data)
}
