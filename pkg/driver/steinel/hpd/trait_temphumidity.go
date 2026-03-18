package hpd

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

type TemperatureSensor struct {
	airtemperaturepb.UnimplementedAirTemperatureApiServer

	logger *zap.Logger

	client *Client

	TemperatureValue *resource.Value
}

var _ sensor = (*TemperatureSensor)(nil)

func newTemperatureSensor(client *Client, logger *zap.Logger) *TemperatureSensor {
	return &TemperatureSensor{
		client:           client,
		logger:           logger,
		TemperatureValue: resource.NewValue(resource.WithInitialValue(&airtemperaturepb.AirTemperature{}), resource.WithNoDuplicates()),
	}
}

func (a *TemperatureSensor) GetAirTemperature(_ context.Context, _ *airtemperaturepb.GetAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	response := SensorResponse{}
	if err := doGetRequest(a.client, &response, "sensor"); err != nil {
		return nil, err
	}
	if err := a.GetUpdate(&response); err != nil {
		return nil, err
	}
	return a.TemperatureValue.Get().(*airtemperaturepb.AirTemperature), nil
}

func (a *TemperatureSensor) PullAirTemperature(request *airtemperaturepb.PullAirTemperatureRequest, server airtemperaturepb.AirTemperatureApi_PullAirTemperatureServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	changes := a.TemperatureValue.Pull(ctx)

	for change := range changes {
		v := change.Value.(*airtemperaturepb.AirTemperature)

		err := server.Send(&airtemperaturepb.PullAirTemperatureResponse{
			Changes: []*airtemperaturepb.PullAirTemperatureResponse_Change{
				{Name: request.GetName(), ChangeTime: timestamppb.New(change.ChangeTime), AirTemperature: v},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *TemperatureSensor) GetUpdate(response *SensorResponse) error {
	humidity := float32(response.Humidity)

	_, err := a.TemperatureValue.Set(&airtemperaturepb.AirTemperature{
		Mode:               0,
		TemperatureGoal:    nil,
		AmbientTemperature: &types.Temperature{ValueCelsius: response.Temperature},
		AmbientHumidity:    &humidity,
		DewPoint:           nil,
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *TemperatureSensor) GetName() string {
	return "Temperature-Humidity"
}
