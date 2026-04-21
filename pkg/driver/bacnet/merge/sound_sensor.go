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
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
)

type soundSensorConfig struct {
	config.Trait

	SoundPressureLevel *config.ValueSource `json:"soundPressureLevel,omitempty"`
}

func readSoundSensorConfig(raw []byte) (cfg soundSensorConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

var _ soundsensorpb.SoundSensorApiServer = (*soundSensor)(nil)

type soundSensor struct {
	soundsensorpb.UnimplementedSoundSensorApiServer

	client     *gobacnet.Client
	known      known.Context
	faultCheck *healthpb.FaultCheck
	logger     *zap.Logger

	model *soundsensorpb.Model
	*soundsensorpb.ModelServer
	config   soundSensorConfig
	pollTask *task.Intermittent
}

func newSoundSensor(client *gobacnet.Client, devices known.Context, faultCheck *healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*soundSensor, error) {
	cfg, err := readSoundSensorConfig(config.Raw)
	if err != nil {
		return nil, err
	}

	model := soundsensorpb.NewModel(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.5), // report sound level changes of 0.5 dB or more
	)))
	s := &soundSensor{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: soundsensorpb.NewModelServer(model),
		config:      cfg,
	}
	s.pollTask = task.NewIntermittent(s.startPoll)
	return s, nil
}

func (s *soundSensor) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(s.config.Name,
		node.HasServer(soundsensorpb.RegisterSoundSensorApiServer, soundsensorpb.SoundSensorApiServer(s)),
		node.HasTrait(soundsensorpb.TraitName),
	)
}

func (s *soundSensor) GetSoundLevel(ctx context.Context, request *soundsensorpb.GetSoundLevelRequest) (*soundsensorpb.SoundLevel, error) {
	_, err := s.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return s.ModelServer.GetSoundLevel(ctx, request)
}

func (s *soundSensor) PullSoundLevel(request *soundsensorpb.PullSoundLevelRequest, server soundsensorpb.SoundSensorApi_PullSoundLevelServer) error {
	_ = s.pollTask.Attach(server.Context())
	return s.ModelServer.PullSoundLevel(request, server)
}

func (s *soundSensor) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "soundSensor", s.config.PollPeriodDuration(), s.config.PollTimeoutDuration(), s.logger, func(ctx context.Context) error {
		_, err := s.pollPeer(ctx)
		return err
	})
}

func (s *soundSensor) pollPeer(ctx context.Context) (*soundsensorpb.SoundLevel, error) {
	data := &soundsensorpb.SoundLevel{}
	var resProcessors []func(response any) error
	var readValues []config.ValueSource

	if s.config.SoundPressureLevel != nil {
		readValues = append(readValues, *s.config.SoundPressureLevel)
		resProcessors = append(resProcessors, func(response any) error {
			v, err := comm.Float32Value(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "soundPressureLevel", Cause: err}
			}
			data.SoundPressureLevel = &v
			return nil
		})
	}

	responses := comm.ReadProperties(ctx, s.client, s.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response)
		if err != nil {
			errs = append(errs, err)
		}
	}
	updateTraitFaultCheck(ctx, s.faultCheck, s.config.Name, soundsensorpb.TraitName, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	return s.model.UpdateSoundLevel(data)
}
