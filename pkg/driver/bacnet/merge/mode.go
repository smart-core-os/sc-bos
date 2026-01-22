package merge

import (
	"context"
	"encoding/json"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/modepb"
)

type modeConfig struct {
	config.Trait
	Modes map[string]modeConfigPoint `json:"modes,omitempty"`
}

type modeConfigPoint struct {
	Title string                  `json:"title,omitempty"`
	Value *config.ValueSource     `json:"value,omitempty"`
	Opts  []modeConfigPointOption `json:"opts,omitempty"`
}

type modeConfigPointOption struct {
	Name  string `json:"name,omitempty"`  // the option as selected by clients
	Value any    `json:"value,omitempty"` // what to write to the point when selecting this option
}

func readModeConfig(raw []byte) (cfg modeConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	// go bacnet doesn't support 64 bit numbers, json uses them by default :(
	for name, point := range cfg.Modes {
		for i, opt := range point.Opts {
			switch v := opt.Value.(type) {
			case float64:
				point.Opts[i].Value = float32(v)
			case int64:
				point.Opts[i].Value = int32(v)
			}
		}
		cfg.Modes[name] = point
	}
	return
}

type mode struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	model *modepb.Model
	*modepb.ModelServer
	infoServer *modepb.InfoServer
	config     modeConfig
	pollTask   *task.Intermittent
}

func newMode(client *gobacnet.Client, devices known.Context, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*mode, error) {
	cfg, err := readModeConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := modepb.NewModel()
	_, _ = model.UpdateModeValues(&traits.ModeValues{}) // clear the default initial value
	t := &mode{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: modepb.NewModelServer(model),
		infoServer:  newModeInfoServer(cfg),
		config:      cfg,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	return t, nil
}

func (t *mode) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(t.config.Name, node.HasTrait(trait.Mode, node.WithClients(
		modepb.WrapApi(t), modepb.WrapInfo(t.infoServer))))
}

func (t *mode) GetModeValues(ctx context.Context, request *traits.GetModeValuesRequest) (*traits.ModeValues, error) {
	_, err := t.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return t.ModelServer.GetModeValues(ctx, request)
}

func (t *mode) UpdateModeValues(ctx context.Context, request *traits.UpdateModeValuesRequest) (*traits.ModeValues, error) {
	type toWrite struct {
		point config.ValueSource
		value any
	}
	var allToWrite []toWrite
	type expected struct {
		name  string
		value string
	}
	var allExpected []expected

	mask := masks.NewResponseFilter(masks.WithFieldMask(request.UpdateMask))
	modelValues := mask.FilterClone(request.ModeValues).(*traits.ModeValues)

	for name, valueName := range modelValues.Values {
		point, ok := t.config.Modes[name]
		if !ok || point.Value == nil {
			return nil, status.Errorf(codes.InvalidArgument, "unknown mode %q", name)
		}
		var value any
		for _, opt := range point.Opts {
			if opt.Name == valueName {
				value = opt.Value
				break
			}
		}
		if value == nil {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported value %q for mode %q", valueName, name)
		}
		allToWrite = append(allToWrite, toWrite{
			point: *point.Value,
			value: value,
		})
		allExpected = append(allExpected, expected{
			name:  name,
			value: valueName,
		})
	}

	var errs []error
	for _, toWrite := range allToWrite {
		err := comm.WriteProperty(ctx, t.client, t.known, toWrite.point, toWrite.value, 0)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, status.Errorf(codes.Internal, "failed to write mode values: %v", multierr.Combine(errs...))
	}

	return pollUntil(ctx, t.config.DefaultRWConsistencyTimeoutDuration(), t.pollPeer, func(values *traits.ModeValues) bool {
		for _, e := range allExpected {
			if values.Values[e.name] != e.value {
				return false
			}
		}
		return true
	})
}

func (t *mode) PullModeValues(request *traits.PullModeValuesRequest, server traits.ModeApi_PullModeValuesServer) error {
	_ = t.pollTask.Attach(server.Context())
	return t.ModelServer.PullModeValues(request, server)
}

func (t *mode) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "mode", t.config.PollPeriodDuration(), t.config.PollTimeoutDuration(), t.logger, func(ctx context.Context) error {
		_, err := t.pollPeer(ctx)
		return err
	})
}

func (t *mode) pollPeer(ctx context.Context) (*traits.ModeValues, error) {
	var readValues []config.ValueSource
	var requestNames []string
	type nameAndPoint struct {
		name  string
		point modeConfigPoint
	}
	var readConfig []nameAndPoint
	for name, point := range t.config.Modes {
		if point.Value == nil {
			continue
		}
		requestNames = append(requestNames, name)
		readValues = append(readValues, *point.Value)
		readConfig = append(readConfig, nameAndPoint{name: name, point: point})
	}
	responses := comm.ReadProperties(ctx, t.client, t.known, readValues...)
	dst := &traits.ModeValues{
		Values: make(map[string]string, len(responses)),
	}
	var errs []error
responses:
	for i, response := range responses {
		cfg := readConfig[i]
		for _, opt := range cfg.point.Opts {
			if valuesEquivalent(opt.Value, response) {
				dst.Values[cfg.name] = opt.Name
				continue responses
			}
		}
		value, err := comm.StringValue(response)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		dst.Values[cfg.name] = value
	}
	updateTraitFaultCheck(t.faultCheck, t.config.Name, trait.Mode, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	return t.model.UpdateModeValues(dst)
}

func newModeInfoServer(cfg modeConfig) *modepb.InfoServer {
	modes := &traits.Modes{}
	for name, point := range cfg.Modes {
		mm := &traits.Modes_Mode{
			Name: name,
		}
		for _, opt := range point.Opts {
			mm.Values = append(mm.Values, &traits.Modes_Value{
				Name: opt.Name,
			})
		}
		modes.Modes = append(modes.Modes, mm)
	}
	return &modepb.InfoServer{
		Modes: &traits.ModesSupport{
			AvailableModes: modes,
		},
	}
}
