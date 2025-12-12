package xovis

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/resource"
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
	gen.UnimplementedUdmiServiceServer

	logger *zap.Logger

	enterLeave      *resource.Value
	occupancy       *resource.Value
	udmiTopicPrefix string
}

func newUdmiServiceServer(logger *zap.Logger, e *resource.Value, o *resource.Value, udmiPrefix string) *udmiServiceServer {
	return &udmiServiceServer{
		logger:          logger,
		enterLeave:      e,
		occupancy:       o,
		udmiTopicPrefix: udmiPrefix,
	}
}

func (u *udmiServiceServer) PullControlTopics(_ *gen.PullControlTopicsRequest, _ gen.UdmiService_PullControlTopicsServer) error {
	// we don't have any control topics
	return status.Error(codes.Unimplemented, "not implemented")
}
func (u *udmiServiceServer) OnMessage(_ context.Context, _ *gen.OnMessageRequest) (*gen.OnMessageResponse, error) {
	// we don't support doing anything here
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (u *udmiServiceServer) GetExportMessage(_ context.Context, _ *gen.GetExportMessageRequest) (*gen.MqttMessage, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (u *udmiServiceServer) PullExportMessages(request *gen.PullExportMessagesRequest, server gen.UdmiService_PullExportMessagesServer) error {
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
				airQuality := change.Value.(*traits.EnterLeaveEvent)
				appendEnterLeaveEventPoints(airQuality, &eventPoints)
			}

		case change, ok := <-occupancyChanges:
			if !ok {
				occupancyChanges = nil
				break
			}
			if change != nil {
				temperature := change.Value.(*traits.Occupancy)
				appendOccupancyEventPoints(temperature, &eventPoints)
			}
		}
		msg := getPointsetMessage(eventPoints)
		eventEnc, err := json.Marshal(msg)
		if err != nil {
			return status.Error(codes.Internal, "failed to encode UDMI message")
		}

		err = server.Send(&gen.PullExportMessagesResponse{
			Name: request.GetName(),
			Message: &gen.MqttMessage{
				Topic:   path.Join(u.udmiTopicPrefix, "events/pointset"),
				Payload: string(eventEnc),
			},
		})
		if err != nil {
			return err
		}
	}
}

func appendEnterLeaveEventPoints(e *traits.EnterLeaveEvent, eventPoints *EventPoints) {
	eventPoints.EnterCount = &EventPoint[int32]{PresentValue: *e.EnterTotal}
	eventPoints.LeaveCount = &EventPoint[int32]{PresentValue: *e.LeaveTotal}
}

func appendOccupancyEventPoints(o *traits.Occupancy, eventPoints *EventPoints) {
	eventPoints.PeopleCount = &EventPoint[int32]{PresentValue: o.PeopleCount}
	eventPoints.OccupancyState = &EventPoint[string]{PresentValue: o.State.String()}
}

func getPointsetMessage(points EventPoints) PointsetEventMessage {
	return PointsetEventMessage{
		Version:       PointsetVersion,
		Timestamp:     time.Now(),
		PartialUpdate: true,
		Points:        points,
	}
}
