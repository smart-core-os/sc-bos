package xovis

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

const PointsetVersion = "1.0.0"

type PointsetEventMessage struct {
	Version       string      `json:"version"`
	Timestamp     time.Time   `json:"timestamp"`
	PartialUpdate bool        `json:"partial_update,omitempty"`
	Points        EventPoints `json:"points"`
}

type EventPoints struct {
	// Generic points
	DeviceType *EventPoint[string] `json:"DeviceType,omitempty"`

	// footfall count related points
	EnterCount *EventPoint[int32] `json:"EnterCount,omitempty"`
	LeaveCount *EventPoint[int32] `json:"LeaveCount,omitempty"`

	// occupancy related points
	PeopleCount    *EventPoint[int32]  `json:"PeopleCount,omitempty"`
	OccupancyState *EventPoint[string] `json:"OccupancyState,omitempty"`
}

type EventPoint[T any] struct {
	PresentValue T `json:"present_value"`
}
type udmiServiceServer struct {
	udmipb.UnimplementedUdmiServiceServer

	logger *zap.Logger

	// enterLeave and occupancy are nil when the device doesn't have that logic configured.
	enterLeave      *resource.Value
	occupancy       *resource.Value
	udmiTopicPrefix string

	// refresh reads every configured logic from the sensor, writing the result
	// through the values above. GetExportMessage calls it so that a pointset it
	// returns always follows a real read.
	refresh func(context.Context) error
	// watch keeps the values above current until ctx is done. Nothing else feeds
	// them, so PullExportMessages has nothing to send without it.
	watch func(context.Context)
}

func newUdmiServiceServer(logger *zap.Logger, e *resource.Value, o *resource.Value, udmiPrefix string, refresh func(context.Context) error, watch func(context.Context)) *udmiServiceServer {
	return &udmiServiceServer{
		logger:          logger,
		enterLeave:      e,
		occupancy:       o,
		udmiTopicPrefix: udmiPrefix,
		refresh:         refresh,
		watch:           watch,
	}
}

func (u *udmiServiceServer) PullControlTopics(_ *udmipb.PullControlTopicsRequest, _ udmipb.UdmiService_PullControlTopicsServer) error {
	// we don't have any control topics
	return status.Error(codes.Unimplemented, "not implemented")
}
func (u *udmiServiceServer) OnMessage(_ context.Context, _ *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	// we don't support doing anything here
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// GetExportMessage reads the sensor and returns a full pointset of every configured
// logic, unlike PullExportMessages which only carries the logic that changed.
// It returns Unavailable if the sensor couldn't be read, or the read's own status,
// such as FailedPrecondition for a logic of the wrong type, or Canceled/DeadlineExceeded
// if ctx ends first.
func (u *udmiServiceServer) GetExportMessage(ctx context.Context, _ *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	if u.refresh == nil {
		return nil, status.Error(codes.Unavailable, "sensor not polled")
	}
	if err := u.refresh(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Unavailable, "read sensor: %v", err)
	}

	eventPoints := EventPoints{
		DeviceType: &EventPoint[string]{PresentValue: DriverName},
	}
	if u.enterLeave != nil {
		appendEnterLeaveEventPoints(u.enterLeave.Get().(*enterleavesensorpb.EnterLeaveEvent), &eventPoints)
	}
	if u.occupancy != nil {
		appendOccupancyEventPoints(u.occupancy.Get().(*occupancysensorpb.Occupancy), &eventPoints)
	}

	eventEnc, err := json.Marshal(newPointsetMessage(eventPoints, false))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode UDMI message")
	}
	return &udmipb.MqttMessage{Topic: u.pointsetTopic(), Payload: string(eventEnc)}, nil
}

func (u *udmiServiceServer) PullExportMessages(request *udmipb.PullExportMessagesRequest, server udmipb.UdmiService_PullExportMessagesServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	var enterLeaveChanges <-chan *resource.ValueChange
	if u.enterLeave != nil {
		enterLeaveChanges = u.enterLeave.Pull(ctx,
			resource.WithUpdatesOnly(true),
		)
	}

	var occupancyChanges <-chan *resource.ValueChange
	if u.occupancy != nil {
		occupancyChanges = u.occupancy.Pull(ctx,
			resource.WithUpdatesOnly(true),
		)
	}

	// Subscribe before starting the poll, which could otherwise change the values
	// before we're listening.
	if u.watch != nil {
		u.watch(ctx)
	}

	for {
		var eventPoints = EventPoints{
			DeviceType: &EventPoint[string]{PresentValue: DriverName},
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case change, ok := <-enterLeaveChanges:
			if !ok {
				enterLeaveChanges = nil
				break
			}
			if change != nil {
				airQuality := change.Value.(*enterleavesensorpb.EnterLeaveEvent)
				appendEnterLeaveEventPoints(airQuality, &eventPoints)
			}

		case change, ok := <-occupancyChanges:
			if !ok {
				occupancyChanges = nil
				break
			}
			if change != nil {
				temperature := change.Value.(*occupancysensorpb.Occupancy)
				appendOccupancyEventPoints(temperature, &eventPoints)
			}
		}
		msg := newPointsetMessage(eventPoints, true)
		eventEnc, err := json.Marshal(msg)
		if err != nil {
			return status.Error(codes.Internal, "failed to encode UDMI message")
		}

		err = server.Send(&udmipb.PullExportMessagesResponse{
			Name: request.GetName(),
			Message: &udmipb.MqttMessage{
				Topic:   u.pointsetTopic(),
				Payload: string(eventEnc),
			},
		})
		if err != nil {
			return err
		}
	}
}

func (u *udmiServiceServer) pointsetTopic() string {
	return path.Join(u.udmiTopicPrefix, "events/pointset")
}

func appendEnterLeaveEventPoints(e *enterleavesensorpb.EnterLeaveEvent, eventPoints *EventPoints) {
	// the totals are unset until the sensor has been read
	if e.EnterTotal != nil {
		eventPoints.EnterCount = &EventPoint[int32]{PresentValue: *e.EnterTotal}
	}
	if e.LeaveTotal != nil {
		eventPoints.LeaveCount = &EventPoint[int32]{PresentValue: *e.LeaveTotal}
	}
}

func appendOccupancyEventPoints(o *occupancysensorpb.Occupancy, eventPoints *EventPoints) {
	eventPoints.PeopleCount = &EventPoint[int32]{PresentValue: o.PeopleCount}
	eventPoints.OccupancyState = &EventPoint[string]{PresentValue: o.State.String()}
}

// newPointsetMessage wraps points in a pointset event. partial marks a message that
// carries only the points that changed, rather than every configured point.
func newPointsetMessage(points EventPoints, partial bool) PointsetEventMessage {
	return PointsetEventMessage{
		Version:       PointsetVersion,
		Timestamp:     time.Now(),
		PartialUpdate: partial,
		Points:        points,
	}
}
