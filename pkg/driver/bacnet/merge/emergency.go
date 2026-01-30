package merge

import (
	"context"
	"encoding/json"
	"math"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/gobacnet/enum/lifesafetystate"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/emergencypb"
)

// AlarmConfig allows configuring a specific bacnet point to raise an Emergency if the
// value of the point read is outside the defined upper and lower bounds.
type AlarmConfig struct {
	config.ValueSource
	// If lower or upper bound is not defined, is defaults to -inf and +inf respectively.
	// You want to set at least 1 of these values, else an emergency will never get raised.
	OkLowerBound *float64 `json:"okLowerBound,omitempty"` // if the point is equal to or greater than this value, it is ok.
	OkUpperBound *float64 `json:"okUpperBound,omitempty"` // if the point is equal to or less than this value, it is ok.
	AlarmReason  string   `json:"alarmReason"`            // the reason of the alarm
}

type emergencyConfig struct {
	config.Trait
	Level       *config.ValueSource `json:"level,omitempty"`
	AlarmConfig *AlarmConfig        `json:"alarmConfig,omitempty"`
}

func readEmergencyConfig(raw []byte) (cfg emergencyConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		return
	}

	if cfg.AlarmConfig != nil {
		if cfg.AlarmConfig.OkLowerBound == nil {
			lowerInf := math.Inf(-1)
			cfg.AlarmConfig.OkLowerBound = &lowerInf
		}
		if cfg.AlarmConfig.OkUpperBound == nil {
			upperInf := math.Inf(1)
			cfg.AlarmConfig.OkUpperBound = &upperInf
		}
	}
	return
}

type emergencyImpl struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	model *emergencypb.MemoryDevice
	traits.EmergencyApiServer
	config   emergencyConfig
	pollTask *task.Intermittent
}

func newEmergency(client *gobacnet.Client, devices known.Context, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*emergencyImpl, error) {
	cfg, err := readEmergencyConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := emergencypb.NewMemoryDevice()
	t := &emergencyImpl{
		client:             client,
		known:              devices,
		faultCheck:         faultCheck,
		logger:             logger,
		model:              model,
		EmergencyApiServer: model,
		config:             cfg,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	return t, nil
}

func (t *emergencyImpl) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "emergency", t.config.PollPeriodDuration(), t.config.PollTimeoutDuration(), t.logger, func(ctx context.Context) error {
		_, err := t.pollPeer(ctx)
		return err
	})
}

func (t *emergencyImpl) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(t.config.Name, node.HasTrait(trait.Emergency, node.WithClients(emergencypb.WrapApi(t))))
}

func (t *emergencyImpl) GetEmergency(ctx context.Context, request *traits.GetEmergencyRequest) (*traits.Emergency, error) {
	_, err := t.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return t.EmergencyApiServer.GetEmergency(ctx, request)
}

func (t *emergencyImpl) UpdateEmergency(ctx context.Context, request *traits.UpdateEmergencyRequest) (*traits.Emergency, error) {
	return traits.UnimplementedEmergencyApiServer{}.UpdateEmergency(ctx, request)
}

func (t *emergencyImpl) PullEmergency(request *traits.PullEmergencyRequest, server traits.EmergencyApi_PullEmergencyServer) error {
	_ = t.pollTask.Attach(server.Context())
	return t.EmergencyApiServer.PullEmergency(request, server)
}

func (t *emergencyImpl) checkValueForEmergency(response any) (*traits.Emergency, error) {
	data := &traits.Emergency{}

	value, err := comm.Float64Value(response)
	if err != nil {
		return nil, comm.ErrReadProperty{Prop: "alarmConfig", Cause: err}
	}

	if value < *t.config.AlarmConfig.OkLowerBound ||
		value > *t.config.AlarmConfig.OkUpperBound {
		data.Reason = t.config.AlarmConfig.AlarmReason
		data.Level = traits.Emergency_EMERGENCY
		return data, nil
	} else {
		data.Level = traits.Emergency_OK
	}
	return data, nil
}

// pollPeer fetches data from the peer device and saves the data locally.
func (t *emergencyImpl) pollPeer(ctx context.Context) (*traits.Emergency, error) {
	data := &traits.Emergency{}
	var resProcessors []func(response any) error
	var readValues []config.ValueSource
	var requestNames []string

	if t.config.Level != nil {
		requestNames = append(requestNames, "level")
		readValues = append(readValues, *t.config.Level)
		resProcessors = append(resProcessors, func(response any) error {
			enum, err := comm.EnumValue(response)
			if err != nil {
				return comm.ErrReadProperty{Prop: "level", Cause: err}
			}
			level := lifesafetystate.LifeSafetyState(enum)
			switch level {
			case lifesafetystate.Quiet:
				data.Level = traits.Emergency_OK
			case lifesafetystate.PreAlarm, lifesafetystate.FaultPreAlarm:
				data.Level = traits.Emergency_WARNING
			default:
				data.Level = traits.Emergency_EMERGENCY
			}

			switch level {
			case lifesafetystate.TestActive, lifesafetystate.TestAlarm, lifesafetystate.TestFault, lifesafetystate.TestFaultAlarm, lifesafetystate.TestSupervisory,
				lifesafetystate.TestOEOAlarm, lifesafetystate.TestOEOEvacuate, lifesafetystate.TestOROPhase1Recall, lifesafetystate.TestOEOUnaffected, lifesafetystate.TestOEOUnavailable:
				data.Drill = true
			default:
				data.Drill = false
			}

			return nil
		})
	}

	if t.config.AlarmConfig != nil {
		requestNames = append(requestNames, "alarmConfig")
		readValues = append(readValues, t.config.AlarmConfig.ValueSource)
		resProcessors = append(resProcessors, func(response any) error {
			if e, err := t.checkValueForEmergency(response); err == nil {
				data = e
			}
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

	updateTraitFaultCheck(ctx, t.faultCheck, t.config.Name, trait.Emergency, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}

	return t.model.UpdateEmergency(ctx, &traits.UpdateEmergencyRequest{Emergency: data})
}
