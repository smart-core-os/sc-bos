package merge

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/onoffpb"
)

type onOffCfg struct {
	config.Trait
	OnOff   *config.ValueSource `json:"onOff,omitempty"`   // the point to read/set onOff
	OnValue *int64              `json:"onValue,omitempty"` // the value that means "on", default 1
}

func readOnOffConfig(raw []byte) (cfg onOffCfg, err error) {
	err = json.Unmarshal(raw, &cfg)

	if cfg.OnValue == nil {
		cfg.OnValue = new(int64)
		*cfg.OnValue = 1
	}

	return
}

type onOff struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	model *onoffpb.Model
	*onoffpb.ModelServer
	config   onOffCfg
	pollTask *task.Intermittent
}

func newOnOff(client *gobacnet.Client, known known.Context, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*onOff, error) {
	cfg, err := readOnOffConfig(config.Raw)
	if err != nil {
		return nil, err
	}

	model := onoffpb.NewModel()
	o := &onOff{
		client:      client,
		known:       known,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: onoffpb.NewModelServer(model),
		config:      cfg,
	}
	o.pollTask = task.NewIntermittent(o.startPoll)
	return o, nil
}

func (o *onOff) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "on off", o.config.PollPeriodDuration(), o.config.PollTimeoutDuration(), o.logger, func(ctx context.Context) error {
		_, err := o.pollPeer(ctx)
		return err
	})
}

func (o *onOff) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(o.config.Name, node.HasTrait(trait.OnOff, node.WithClients(onoffpb.WrapApi(o))))
}

func (o *onOff) GetOnOff(ctx context.Context, request *traits.GetOnOffRequest) (*traits.OnOff, error) {
	_, err := o.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return o.ModelServer.GetOnOff(ctx, request)
}

func (o *onOff) UpdateOnOff(ctx context.Context, request *traits.UpdateOnOffRequest) (*traits.OnOff, error) {

	toSet := request.GetOnOff()
	if toSet == nil || toSet.State == traits.OnOff_STATE_UNSPECIFIED {
		o.logger.Error("UpdateOnOff missing or unspecified OnOffs")
		return nil, errors.New("missing or unspecified OnOffs")
	}

	if o.config.OnOff != nil {
		toWrite := int64(0)
		if toSet.State == traits.OnOff_ON {
			toWrite = *o.config.OnValue
		}
		err := comm.WriteProperty(ctx, o.client, o.known, *o.config.OnOff, toWrite, 0)
		if err != nil {
			o.logger.Error("WriteProperty SetPointLow", zap.Error(err))
			return nil, err
		}
	}

	return pollUntil(ctx, o.config.DefaultRWConsistencyTimeoutDuration(), o.pollPeer, func(onOff *traits.OnOff) bool {
		return proto.Equal(onOff, toSet)
	})
}

func (o *onOff) PullOnOff(request *traits.PullOnOffRequest, server traits.OnOffApi_PullOnOffServer) error {
	_ = o.pollTask.Attach(server.Context())
	return o.ModelServer.PullOnOff(request, server)
}

func (o *onOff) pollPeer(ctx context.Context) (*traits.OnOff, error) {
	data := &traits.OnOff{}

	var resProcessors []func(response any) error
	var readValues []config.ValueSource
	var requestNames []string

	if o.config.OnOff != nil {
		requestNames = append(requestNames, "onOff")
		readValues = append(readValues, *o.config.OnOff)
		resProcessors = append(resProcessors, func(response any) error {
			value, err := comm.IntValue(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "onOff", Cause: err}
			}
			if value == *o.config.OnValue {
				data.State = traits.OnOff_ON
			} else {
				data.State = traits.OnOff_OFF
			}
			return nil
		})
	}
	responses := comm.ReadProperties(ctx, o.client, o.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(o.faultCheck, o.config.Name, trait.OnOff, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	return o.model.UpdateOnOff(data)
}
