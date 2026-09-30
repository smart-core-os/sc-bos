package hpd

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
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

	// air quality related points
	Co2Level      *EventPoint[float32] `json:"Co2Level,omitempty"`
	VocLevel      *EventPoint[float32] `json:"VocLevel,omitempty"`
	AirPressure   *EventPoint[float32] `json:"AirPressure,omitempty"`
	InfectionRisk *EventPoint[float32] `json:"InfectionRisk,omitempty"`
	IAQ           *EventPoint[float32] `json:"IAQ,omitempty"`

	// TemperatureValue related points
	Temperature *EventPoint[float64] `json:"Temperature,omitempty"`
	Humidity    *EventPoint[float32] `json:"Humidity,omitempty"`

	// OccupancyValue related points
	PeopleCount    *EventPoint[int32]  `json:"PeopleCount,omitempty"`
	OccupancyState *EventPoint[string] `json:"OccupancyState,omitempty"`
}

type EventPoint[T any] struct {
	PresentValue T `json:"present_value"`
}

type UdmiServiceServer struct {
	udmipb.UnimplementedUdmiServiceServer

	logger *zap.Logger

	// airQuality is nil when the device model has no air quality module.
	airQuality      *resource.Value
	occupancySensor *resource.Value
	tempHumidity    *resource.Value
	udmiTopicPrefix string

	// refresh reads the device, writing the result through the values above. The
	// values keep their last reading when a poll fails, so GetExportMessage calls
	// this rather than trusting them: a pointset it returns always follows a real read.
	refresh func(context.Context) error
}

func newUdmiServiceServer(logger *zap.Logger, aq *resource.Value, o *resource.Value, t *resource.Value, udmiPrefix string, refresh func(context.Context) error) *UdmiServiceServer {
	return &UdmiServiceServer{
		logger:          logger,
		airQuality:      aq,
		occupancySensor: o,
		tempHumidity:    t,
		udmiTopicPrefix: udmiPrefix,
		refresh:         refresh,
	}
}

func (u *UdmiServiceServer) PullControlTopics(_ *udmipb.PullControlTopicsRequest, _ udmipb.UdmiService_PullControlTopicsServer) error {
	// we don't have any control topics
	return status.Error(codes.Unimplemented, "not implemented")
}
func (u *UdmiServiceServer) OnMessage(_ context.Context, _ *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	// we don't support doing anything here
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// GetExportMessage reads the device and returns a full pointset of every point it
// has, unlike PullExportMessages which only carries the points that changed.
// It returns Unavailable if the device couldn't be read, or Canceled/DeadlineExceeded
// if ctx ends first.
func (u *UdmiServiceServer) GetExportMessage(ctx context.Context, _ *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	if u.refresh == nil {
		return nil, status.Error(codes.Unavailable, "device not polled")
	}
	if err := u.refresh(ctx); err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		return nil, status.Errorf(codes.Unavailable, "read device: %v", err)
	}

	eventPoints := EventPoints{
		DeviceType: &EventPoint[string]{PresentValue: DriverName},
	}
	if u.airQuality != nil {
		appendAirQualityEventPoints(u.airQuality.Get().(*airqualitysensorpb.AirQuality), &eventPoints)
	}
	appendTempHumidityEventPoints(u.tempHumidity.Get().(*airtemperaturepb.AirTemperature), &eventPoints)
	appendOccupancyEventPoints(u.occupancySensor.Get().(*occupancysensorpb.Occupancy), &eventPoints)

	eventEnc, err := json.Marshal(newPointsetMessage(eventPoints, false))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode UDMI message")
	}
	return &udmipb.MqttMessage{Topic: u.pointsetTopic(), Payload: string(eventEnc)}, nil
}

func (u *UdmiServiceServer) PullExportMessages(request *udmipb.PullExportMessagesRequest, server udmipb.UdmiService_PullExportMessagesServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	var airQualityChanges <-chan *resource.ValueChange
	if u.airQuality != nil {
		airQualityChanges = u.airQuality.Pull(ctx,
			resource.WithUpdatesOnly(true),
		)
	}

	temperatureChanges := u.tempHumidity.Pull(ctx,
		resource.WithUpdatesOnly(true),
	)

	occupancyChanges := u.occupancySensor.Pull(ctx,
		resource.WithUpdatesOnly(true),
	)

	for {
		var eventPoints = EventPoints{
			DeviceType: &EventPoint[string]{PresentValue: DriverName},
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case change := <-airQualityChanges:
			if change != nil {
				airQuality := change.Value.(*airqualitysensorpb.AirQuality)
				appendAirQualityEventPoints(airQuality, &eventPoints)
			}

		case change := <-temperatureChanges:
			if change != nil {
				temperature := change.Value.(*airtemperaturepb.AirTemperature)
				appendTempHumidityEventPoints(temperature, &eventPoints)
			}
		case change := <-occupancyChanges:
			if change != nil {
				occupancy := change.Value.(*occupancysensorpb.Occupancy)
				appendOccupancyEventPoints(occupancy, &eventPoints)
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

func (u *UdmiServiceServer) pointsetTopic() string {
	return path.Join(u.udmiTopicPrefix, "events/pointset")
}

// The append functions skip fields that haven't been read. The values start out
// empty, and publishing a zero would claim a reading the device never gave.

func appendAirQualityEventPoints(aq *airqualitysensorpb.AirQuality, eventPoints *EventPoints) {
	if aq.CarbonDioxideLevel != nil {
		eventPoints.Co2Level = &EventPoint[float32]{PresentValue: *aq.CarbonDioxideLevel}
	}
	if aq.VolatileOrganicCompounds != nil {
		eventPoints.VocLevel = &EventPoint[float32]{PresentValue: *aq.VolatileOrganicCompounds}
	}

	if aq.Score != nil {
		eventPoints.IAQ = &EventPoint[float32]{PresentValue: *aq.Score}
	}

	if aq.InfectionRisk != nil {
		eventPoints.InfectionRisk = &EventPoint[float32]{PresentValue: *aq.InfectionRisk}
	}

	if aq.AirPressure != nil {
		eventPoints.AirPressure = &EventPoint[float32]{PresentValue: *aq.AirPressure}
	}
}

func appendTempHumidityEventPoints(a *airtemperaturepb.AirTemperature, eventPoints *EventPoints) {
	if a.AmbientTemperature != nil {
		eventPoints.Temperature = &EventPoint[float64]{PresentValue: a.AmbientTemperature.ValueCelsius}
	}
	if a.AmbientHumidity != nil {
		eventPoints.Humidity = &EventPoint[float32]{PresentValue: *a.AmbientHumidity}
	}
}

func appendOccupancyEventPoints(o *occupancysensorpb.Occupancy, eventPoints *EventPoints) {
	eventPoints.PeopleCount = &EventPoint[int32]{PresentValue: o.PeopleCount}
	eventPoints.OccupancyState = &EventPoint[string]{PresentValue: o.State.String()}
}

// newPointsetMessage wraps points in a pointset event. partial marks a message that
// carries only the points that changed, rather than every point the device has.
func newPointsetMessage(points EventPoints, partial bool) PointsetEventMessage {
	return PointsetEventMessage{
		Version:       PointsetVersion,
		Timestamp:     time.Now(),
		PartialUpdate: partial,
		Points:        points,
	}
}
