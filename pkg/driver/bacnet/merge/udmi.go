package merge

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

const UdmiMergeName = "udmi"

// DefaultEventsTopicSuffix is appended to TopicPrefix when no TopicSuffix is configured.
// The UDMI spec (6.4) uses "/events/pointset"; this default preserves the legacy
// "/event/pointset/points" topic for backward compatibility.
const DefaultEventsTopicSuffix = "/event/pointset/points"

// Default state/metadata topic suffixes, matching the rest of the UDMI
// deployment (validator accepts both "/metadata" and "/metadata.json").
const (
	DefaultStateTopicSuffix    = "/state"
	DefaultMetadataTopicSuffix = "/metadata.json"
)

type UdmiMergeConfig struct {
	config.Trait
	TopicPrefix string `json:"topicPrefix,omitempty"`
	// TopicSuffix is appended to TopicPrefix to form the MQTT topic for pointset events.
	// Defaults to DefaultEventsTopicSuffix. Set to "/events/pointset" for UDMI spec compliance.
	TopicSuffix *string `json:"topicSuffix,omitempty"`
	// UDMIVersion is the schema version stamped into the pointset envelope.
	// Only used when the envelope is emitted (see EmitEnvelope); defaults to udmi.PointsetVersion.
	UDMIVersion string `json:"udmiVersion,omitempty"`
	// EmitStateMetadata enables publishing UDMI state (state.json) and metadata
	// (metadata.json) messages alongside pointset events. Off by default.
	EmitStateMetadata bool `json:"emitStateMetadata,omitempty"`
	// StateTopicSuffix / MetadataTopicSuffix override the default topic suffixes
	// used when EmitStateMetadata is set.
	StateTopicSuffix    *string `json:"stateTopicSuffix,omitempty"`
	MetadataTopicSuffix *string `json:"metadataTopicSuffix,omitempty"`
	// Hardware overrides the make/model reported in state.system.hardware.
	Hardware *UdmiHardware                  `json:"hardware,omitempty"`
	Points   map[string]*config.ValueSource `json:"points"`
}

// UdmiHardware configures the state.system.hardware make/model of a device.
type UdmiHardware struct {
	Make  string `json:"make,omitempty"`
	Model string `json:"model,omitempty"`
}

// EventsTopicSuffix returns the configured topic suffix or the default if unset.
func (c UdmiMergeConfig) EventsTopicSuffix() string {
	if c.TopicSuffix == nil {
		return DefaultEventsTopicSuffix
	}
	return *c.TopicSuffix
}

// StateTopic / MetadataTopic return the full MQTT topics for state and metadata.
func (c UdmiMergeConfig) StateTopic() string {
	suffix := DefaultStateTopicSuffix
	if c.StateTopicSuffix != nil {
		suffix = *c.StateTopicSuffix
	}
	return c.TopicPrefix + suffix
}

func (c UdmiMergeConfig) MetadataTopic() string {
	suffix := DefaultMetadataTopicSuffix
	if c.MetadataTopicSuffix != nil {
		suffix = *c.MetadataTopicSuffix
	}
	return c.TopicPrefix + suffix
}

// hardware returns the make/model for state.system.hardware, applying defaults
// where unconfigured (state_system.json requires both to be non-empty).
func (c UdmiMergeConfig) hardware() udmi.SystemHardware {
	h := udmi.SystemHardware{Make: "Vanti", Model: "Smart Core BACnet"}
	if c.Hardware != nil {
		if c.Hardware.Make != "" {
			h.Make = c.Hardware.Make
		}
		if c.Hardware.Model != "" {
			h.Model = c.Hardware.Model
		}
	}
	return h
}

// EmitEnvelope reports whether pointset payloads should be wrapped in the UDMI
// {timestamp, version, points} envelope. The legacy DefaultEventsTopicSuffix keeps
// the bare points-map shape for backward compatibility; the UDMI-spec "/events/pointset"
// topic (and any other explicit suffix) gets the compliant envelope.
func (c UdmiMergeConfig) EmitEnvelope() bool {
	return c.EventsTopicSuffix() != DefaultEventsTopicSuffix
}

// Version returns the UDMI schema version to stamp into the envelope, or the
// default when unconfigured.
func (c UdmiMergeConfig) Version() string {
	if c.UDMIVersion != "" {
		return c.UDMIVersion
	}
	return udmi.PointsetVersion
}

func readUdmiMergeConfig(raw []byte) (cfg UdmiMergeConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

// udmiMerge implements the UdmiService and will merge multiple BACnet objects into one UDMI payload
// BACnet objects are polled for changes, and any changes sent as UDMI events
// control is implemented via OnMessage, only points present in the config are controllable.
type udmiMerge struct {
	udmipb.UnimplementedUdmiServiceServer
	client     *gobacnet.Client
	known      known.Context
	faultCheck *healthpb.FaultCheck
	logger     *zap.Logger

	config UdmiMergeConfig
	bus    minibus.Bus[*udmipb.PullExportMessagesResponse]

	pollTask *task.Intermittent
	// protect the points value
	pointsLock sync.Mutex
	points     udmi.PointsEvent

	// operational tracks whether the most recent poll reached the device; it
	// drives the UDMI state message's system.operation.operational field.
	operational atomic.Bool
}

func newUdmiMerge(client *gobacnet.Client, devices known.Context, faultCheck *healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*udmiMerge, error) {
	cfg, err := readUdmiMergeConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	f := &udmiMerge{
		client:     client,
		known:      devices,
		faultCheck: faultCheck,
		config:     cfg,
		logger:     logger,
	}
	f.pollTask = task.NewIntermittent(f.startPoll)
	return f, nil
}

func (f *udmiMerge) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(f.config.Name,
		node.HasServer(udmipb.RegisterUdmiServiceServer, udmipb.UdmiServiceServer(f)),
		node.HasTrait(udmipb.TraitName),
	)
}

func (f *udmiMerge) PullControlTopics(request *udmipb.PullControlTopicsRequest, server udmipb.UdmiService_PullControlTopicsServer) error {
	err := server.Send(&udmipb.PullControlTopicsResponse{
		Name:   f.config.Name,
		Topics: []string{f.config.TopicPrefix + "/config"},
	})
	if err != nil {
		return err
	}
	ctx := server.Context()
	<-ctx.Done()
	return ctx.Err()
}

func (f *udmiMerge) OnMessage(ctx context.Context, request *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	if request.Message == nil {
		return nil, status.Error(codes.InvalidArgument, "no message")
	}
	var msg udmi.ConfigMessage
	err := json.Unmarshal([]byte(request.Message.Payload), &msg)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config message: %s", err)
	}
	for point, value := range msg.PointSet.Points {
		if cfg, ok := f.config.Points[point]; ok {
			set := value.SetValue
			if floatValue, isFloat := value.SetValue.(float64); isFloat {
				// let's assume for now values shouldn't be float64, true for the YABE room simulator at least
				set = float32(floatValue)
			}
			err = comm.WriteProperty(ctx, f.client, f.known, *cfg, set, 0)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to write point %s: %s", point, err)
			}
		}
	}
	return &udmipb.OnMessageResponse{Name: f.config.Name}, nil
}

func (f *udmiMerge) GetExportMessage(ctx context.Context, request *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	pollCtx, cleanup := context.WithTimeout(ctx, f.config.PollTimeoutDuration()/4)
	defer cleanup()
	events := f.bus.Listen(pollCtx)
	_ = f.pollTask.Attach(pollCtx)
	select {
	case <-pollCtx.Done():
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		f.pointsLock.Lock()
		points := f.points
		f.pointsLock.Unlock()
		if len(points) == 0 {
			return nil, status.Error(codes.Unavailable, "no recent events")
		}
		return f.pointsToPointSet(points)
	case msg := <-events:
		return msg.Message, nil
	}
}

func (f *udmiMerge) PullExportMessages(request *udmipb.PullExportMessagesRequest, server udmipb.UdmiService_PullExportMessagesServer) error {
	events := f.bus.Listen(server.Context())
	_ = f.pollTask.Attach(server.Context())

	// Announce the device model and current status up front, so a (re)connecting
	// subscriber gets metadata and state without waiting for the next change.
	if f.config.EmitStateMetadata {
		for _, build := range []func() (*udmipb.MqttMessage, error){f.metadataMessage, f.stateMessage} {
			msg, err := build()
			if err != nil {
				return err
			}
			if err := server.Send(&udmipb.PullExportMessagesResponse{Name: request.Name, Message: msg}); err != nil {
				return err
			}
		}
	}

	// initial value
	if request.IncludeLast {
		f.pointsLock.Lock()
		points := f.points
		f.pointsLock.Unlock()
		if len(points) > 0 {
			msg, err := f.pointsToPointSet(points)
			if err != nil {
				return err
			}
			err = server.Send(&udmipb.PullExportMessagesResponse{
				Name:    request.Name,
				Message: msg,
			})
			if err != nil {
				return err
			}
		}
	}
	for msg := range events {
		err := server.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *udmiMerge) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "udmi", f.config.PollPeriodDuration(), f.config.PollTimeoutDuration(), f.logger, f.pollPeer)
}

// pollPeer fetches data from the peer device, save locally, and fire a change if there is one
func (f *udmiMerge) pollPeer(ctx context.Context) error {
	events := make(udmi.PointsEvent)
	var errs []error
	requestValues := make([]config.ValueSource, 0, len(f.config.Points))
	keys := make([]string, 0, len(f.config.Points))
	for key, cfg := range f.config.Points {
		requestValues = append(requestValues, *cfg)
		keys = append(keys, key)
	}
	for i, result := range comm.ReadPropertiesChunked(ctx, f.client, f.known, f.config.ChunkSize, requestValues...) {
		switch e := result.(type) {
		case error:
			errs = append(errs, comm.ErrReadProperty{Prop: keys[i], Cause: e})
		default:
			events[keys[i]] = udmi.PointValue{PresentValue: e}
		}
	}

	updateTraitFaultCheck(ctx, f.faultCheck, f.config.Name, udmipb.TraitName, errs)

	// Re-publish state only when operational changes, mirroring the pointset
	// hasUpdate behaviour below. Metadata is effectively static and isn't
	// re-published per poll at all; (re)connecting subscribers get both metadata
	// and state up front from PullExportMessages. This runs before the all-failed
	// return below so an operational=false transition is reported when the device
	// becomes unreachable.
	allFailed := len(errs) == len(f.config.Points)
	operational := !allFailed
	prevOperational := f.operational.Swap(operational)
	if f.config.EmitStateMetadata && operational != prevOperational {
		f.sendState(ctx)
	}

	if allFailed {
		err := multierr.Combine(errs...)
		return err
	}
	if len(errs) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	f.pointsLock.Lock()
	isEqual := f.points.Equal(events)
	hasUpdate := !isEqual
	if hasUpdate {
		f.points = events
	}
	f.pointsLock.Unlock()
	if hasUpdate {
		// send the update
		msg, err := f.pointsToPointSet(events)
		if err != nil {
			return err
		}
		f.bus.Send(ctx, &udmipb.PullExportMessagesResponse{
			Name:    f.config.Name,
			Message: msg,
		})
	}
	return nil
}

func sanitise(points udmi.PointsEvent) {
	for k, v := range points {
		if pv, ok := v.PresentValue.(float64); ok {
			if math.IsNaN(pv) || math.IsInf(pv, 0) {
				points[k] = udmi.PointValue{PresentValue: nil}
			}
		}
	}
}

func (f *udmiMerge) pointsToPointSet(points udmi.PointsEvent) (*udmipb.MqttMessage, error) {

	sanitise(points)

	// On the UDMI-spec topic, wrap the points in the {timestamp, version, points}
	// envelope events_pointset.json requires; the bare points map fails schema
	// validation. The legacy topic keeps the bare map for backward compatibility.
	var payload any = points
	if f.config.EmitEnvelope() {
		payload = udmi.PointsetEvent{
			Timestamp: time.Now().UTC(),
			Version:   f.config.Version(),
			Points:    points,
		}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &udmipb.MqttMessage{
		Topic:   f.config.TopicPrefix + f.config.EventsTopicSuffix(),
		Payload: string(b),
	}, nil
}

// site extracts the BDNS site code from a "<client>/<site>/<system>/<device>" topic
// prefix, or "" if the prefix doesn't have that shape.
func site(topicPrefix string) string {
	parts := strings.Split(topicPrefix, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// metadataMessage builds the UDMI metadata (metadata.json) message describing
// the device's model.
func (f *udmiMerge) metadataMessage() (*udmipb.MqttMessage, error) {
	sys := udmi.MetadataSystem{Name: f.config.Name}
	if s := site(f.config.TopicPrefix); s != "" {
		sys.Location = &udmi.MetadataLocation{Site: s}
	}
	b, err := json.Marshal(udmi.MetadataEvent{
		Timestamp: time.Now().UTC(),
		Version:   f.config.Version(),
		System:    sys,
	})
	if err != nil {
		return nil, err
	}
	return &udmipb.MqttMessage{Topic: f.config.MetadataTopic(), Payload: string(b)}, nil
}

// stateMessage builds the UDMI state (state.json) message reporting the device's
// current operational status (true once a poll has reached the device).
func (f *udmiMerge) stateMessage() (*udmipb.MqttMessage, error) {
	now := time.Now().UTC()
	b, err := json.Marshal(udmi.StateEvent{
		Timestamp: now,
		Version:   f.config.Version(),
		System: udmi.StateSystem{
			// serial_no is left empty: a merge device is a logical grouping of
			// BACnet objects with no single hardware serial.
			LastConfig: now,
			Hardware:   f.config.hardware(),
			Software:   map[string]string{"driver": "bacnet"},
			Operation:  udmi.SystemOperation{Operational: f.operational.Load()},
		},
	})
	if err != nil {
		return nil, err
	}
	return &udmipb.MqttMessage{Topic: f.config.StateTopic(), Payload: string(b)}, nil
}

// sendState publishes the current state message to the export bus. Called from
// pollPeer only when system.operation.operational changes, so existing
// subscribers see the device go (un)reachable without a message every poll.
func (f *udmiMerge) sendState(ctx context.Context) {
	msg, err := f.stateMessage()
	if err != nil {
		f.logger.Warn("failed to build udmi state message", zap.Error(err))
		return
	}
	f.bus.Send(ctx, &udmipb.PullExportMessagesResponse{Name: f.config.Name, Message: msg})
}

func (f *udmiMerge) PollTask() *task.Intermittent { return f.pollTask }
