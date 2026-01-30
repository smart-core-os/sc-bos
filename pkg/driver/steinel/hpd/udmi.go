package hpd

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
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

	airQuality      *resource.Value
	occupancySensor *resource.Value
	tempHumidity    *resource.Value
	udmiTopicPrefix string
}

func newUdmiServiceServer(logger *zap.Logger, aq *resource.Value, o *resource.Value, t *resource.Value, udmiPrefix string) *UdmiServiceServer {
	return &UdmiServiceServer{
		logger:          logger,
		airQuality:      aq,
		occupancySensor: o,
		tempHumidity:    t,
		udmiTopicPrefix: udmiPrefix,
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

func (u *UdmiServiceServer) GetExportMessage(_ context.Context, _ *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (u *UdmiServiceServer) PullExportMessages(request *udmipb.PullExportMessagesRequest, server udmipb.UdmiService_PullExportMessagesServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	airQualityChanges := u.airQuality.Pull(ctx,
		resource.WithUpdatesOnly(true),
	)

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

		case change, _ := <-airQualityChanges:
			if change != nil {
				airQuality := change.Value.(*traits.AirQuality)
				appendAirQualityEventPoints(airQuality, &eventPoints)
			}

		case change, _ := <-temperatureChanges:
			if change != nil {
				temperature := change.Value.(*traits.AirTemperature)
				appendTempHumidityEventPoints(temperature, &eventPoints)
			}
		case change, _ := <-occupancyChanges:
			if change != nil {
				occupancy := change.Value.(*traits.Occupancy)
				appendOccupancyEventPoints(occupancy, &eventPoints)
			}
		}
		msg := getPointsetMessage(eventPoints)
		eventEnc, err := json.Marshal(msg)
		if err != nil {
			return status.Error(codes.Internal, "failed to encode UDMI message")
		}

		err = server.Send(&udmipb.PullExportMessagesResponse{
			Name: request.GetName(),
			Message: &udmipb.MqttMessage{
				Topic:   path.Join(u.udmiTopicPrefix, "events/pointset"),
				Payload: string(eventEnc),
			},
		})
		if err != nil {
			return err
		}
	}
}

func appendAirQualityEventPoints(aq *traits.AirQuality, eventPoints *EventPoints) {

	eventPoints.Co2Level = &EventPoint[float32]{PresentValue: *aq.CarbonDioxideLevel}
	eventPoints.VocLevel = &EventPoint[float32]{PresentValue: *aq.VolatileOrganicCompounds}

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

func appendTempHumidityEventPoints(a *traits.AirTemperature, eventPoints *EventPoints) {
	eventPoints.Temperature = &EventPoint[float64]{PresentValue: a.AmbientTemperature.ValueCelsius}
	eventPoints.Humidity = &EventPoint[float32]{PresentValue: *a.AmbientHumidity}
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
