package merge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/pborman/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/securityevent"
	gen_securityevent "github.com/smart-core-os/sc-bos/pkg/gentrait/securityevent"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

type securityEventSource struct {
	*config.ValueSource
	// description of the security event, must be set to make sense of the event
	// if IsString is true, this description acts as a prefix to the value source.
	Description string `json:"description"`
	// You want to set at least 1 of these OkBound values, else a security event will never get raised.
	OkLowerBound *float64 `json:"okLowerBound,omitempty"` // if the point is equal to or greater than this value, it is ok.
	OkUpperBound *float64 `json:"okUpperBound,omitempty"` // if the point is equal to or less than this value, it is ok.

	Actor     *string                                  `json:"actor,omitempty"`     // Optional. Actor of the security event, e.g. "John Doe"
	EventType *securityeventpb.SecurityEvent_EventType `json:"eventType,omitempty"` // Optional. the type of event, must be one of securityeventpb.SecurityEvent_EventType
	Priority  *int32                                   `json:"priority,omitempty"`  // Optional. Priority of the security event, lower is more important
	Source    *string                                  `json:"source,omitempty"`    // Optional. Source of the security event, e.g. "Door 1"

	IsString      bool   `json:"isString,omitempty"`      // Optional. If true, the value source is a string, otherwise it is a float64.
	OkStringValue string `json:"okStringValue,omitempty"` // if IsString is true, this is the value that is considered OK i.e. will deactivate the security event.
}

type securityEvent struct {
	cfg      securityEventSource
	IsActive bool

	previousValue any // used to track the previous value of the security event source, to avoid duplicate events
}

type securityEventConfig struct {
	config.Trait
	SecurityEventSources []*securityEventSource `json:"securityEventSources"`
}

func readSecurityEventConfig(raw []byte) (cfg securityEventConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		return
	}

	if cfg.SecurityEventSources == nil || len(cfg.SecurityEventSources) == 0 {
		return cfg, errors.New("no security events configured")
	}

	for _, se := range cfg.SecurityEventSources {
		if se == nil {
			return cfg, errors.New("nil security event source found in config")
		}
		if se.ValueSource == nil {
			return cfg, errors.New("no value source provided for security event")
		}
		if se.Description == "" {
			return cfg, errors.New("no description provided for security event")
		}
		if se.OkLowerBound == nil {
			lowerInf := math.Inf(-1)
			se.OkLowerBound = &lowerInf
		}
		if se.OkUpperBound == nil {
			upperInf := math.Inf(1)
			se.OkUpperBound = &upperInf
		}
	}

	return
}

type securityEventImpl struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *gen_healthpb.FaultCheck
	logger     *zap.Logger

	model *securityevent.Model
	*securityevent.ModelServer
	config   securityEventConfig
	events   []*securityEvent
	pollTask *task.Intermittent
}

func newSecurityEvent(client *gobacnet.Client, devices known.Context, faultCheck *gen_healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*securityEventImpl, error) {
	cfg, err := readSecurityEventConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := securityevent.NewModel()
	var events []*securityEvent
	for _, se := range cfg.SecurityEventSources {
		e := &securityEvent{
			cfg:      *se,
			IsActive: false,
		}
		events = append(events, e)
	}
	t := &securityEventImpl{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: securityevent.NewModelServer(model),
		config:      cfg,
		events:      events,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	return t, nil
}

func (s *securityEventImpl) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "securityevent", s.config.PollPeriodDuration(), s.config.PollTimeoutDuration(), s.logger, func(ctx context.Context) error {
		return s.pollPeer(ctx)
	})
}

func (s *securityEventImpl) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(s.config.Name, node.HasTrait(securityevent.TraitName, node.WithClients(securityeventpb.WrapApi(s))))
}

func (s *securityEventImpl) ListSecurityEvents(ctx context.Context, request *securityeventpb.ListSecurityEventsRequest) (*securityeventpb.ListSecurityEventsResponse, error) {
	err := s.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return s.ModelServer.ListSecurityEvents(ctx, request)
}

func (s *securityEventImpl) PullSecurityEvents(request *securityeventpb.PullSecurityEventsRequest, server securityeventpb.SecurityEventApi_PullSecurityEventsServer) error {
	_ = s.pollTask.Attach(server.Context())
	return s.ModelServer.PullSecurityEvents(request, server)
}

func (s *securityEventImpl) pollPeer(ctx context.Context) error {
	var data []*securityeventpb.SecurityEvent
	var resProcessors []func(response any) error
	var readValues []config.ValueSource
	var requestNames []string

	for _, se := range s.events {
		se := se
		requestNames = append(requestNames, se.cfg.ValueSource.String())
		readValues = append(readValues, *se.cfg.ValueSource)
		resProcessors = append(resProcessors, func(response any) error {
			event, err := se.checkResponseForSecurityEvent(response)
			if err != nil {
				return err
			}
			if event != nil {
				data = append(data, event)
			}
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
	updateTraitFaultCheck(ctx, s.faultCheck, s.config.Name, gen_securityevent.TraitName, errs)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	for _, se := range data {
		_, err := s.model.AddSecurityEvent(se)
		if err != nil {
			errs = append(errs, err)
			s.logger.Warn("failed to add security event", zap.Error(err), zap.String("eventId", se.Id), zap.String("description", se.Description))
		}
	}
	return errors.Join(errs...)
}

func (se *securityEvent) checkResponseForSecurityEvent(response any) (*securityeventpb.SecurityEvent, error) {
	data := &securityeventpb.SecurityEvent{}

	if se.cfg.IsString {
		value, err := comm.StringValue(response)
		if err != nil {
			return nil, comm.ErrReadProperty{Prop: "securityEvent", Cause: err}
		}

		if value == se.cfg.OkStringValue {
			se.IsActive = false
			return nil, nil // no event if the value is empty
		}

		if se.IsActive {
			// if the event is already active, we only want to update it if the value has changed
			if value == se.previousValue {
				return nil, nil
			}
			se.previousValue = value
		}

		se.IsActive = true

		se.cfgDefaults(data)

		if se.cfg.Description != "" {
			data.Description = fmt.Sprintf("%s: %s", se.cfg.Description, value)
		} else {
			data.Description = value
		}

		return data, nil
	}

	value, err := comm.Float64Value(response)
	if err != nil {
		return nil, comm.ErrReadProperty{Prop: "securityEvent", Cause: err}
	}

	if value < *se.cfg.OkLowerBound ||
		value > *se.cfg.OkUpperBound {
		// we only want to add a new security event if it wasn't active on the last poll
		if !se.IsActive {
			se.IsActive = true
			se.cfgDefaults(data)
			data.Description = se.cfg.Description
			return data, nil
		}
	} else {
		se.IsActive = false
	}
	return nil, nil
}

func (se *securityEvent) cfgDefaults(data *securityeventpb.SecurityEvent) {
	data.Id = uuid.New()
	data.SecurityEventTime = timestamppb.Now() // not strictly true but as long as the bacnet poll is not too slow, this should be fine

	if se.cfg.Actor != nil {
		data.Actor = &actorpb.Actor{
			Name: *se.cfg.Actor,
		}
	}

	if se.cfg.EventType != nil {
		data.EventType = *se.cfg.EventType
	}

	if se.cfg.Priority != nil {
		data.Priority = *se.cfg.Priority
	}

	if se.cfg.Source != nil {
		data.Source = &securityeventpb.SecurityEvent_Source{
			Name: *se.cfg.Source,
		}
	}
}
