package hpd

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type TemperatureSensor struct {
	traits.UnimplementedAirTemperatureApiServer

	logger *zap.Logger

	client *Client

	TemperatureValue *resource.Value
}

var _ sensor = (*TemperatureSensor)(nil)

func newTemperatureSensor(client *Client, logger *zap.Logger) *TemperatureSensor {
	return &TemperatureSensor{
		client:           client,
		logger:           logger,
		TemperatureValue: resource.NewValue(resource.WithInitialValue(&traits.AirTemperature{}), resource.WithNoDuplicates()),
	}
}

func (a *TemperatureSensor) GetAirTemperature(_ context.Context, _ *traits.GetAirTemperatureRequest) (*traits.AirTemperature, error) {
	response := SensorResponse{}
	if err := doGetRequest(a.client, &response, "sensor"); err != nil {
		return nil, err
	}
	if err := a.GetUpdate(&response); err != nil {
		return nil, err
	}
	return a.TemperatureValue.Get().(*traits.AirTemperature), nil
}

func (a *TemperatureSensor) PullAirTemperature(request *traits.PullAirTemperatureRequest, server traits.AirTemperatureApi_PullAirTemperatureServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	changes := a.TemperatureValue.Pull(ctx)

	for change := range changes {
		v := change.Value.(*traits.AirTemperature)

		err := server.Send(&traits.PullAirTemperatureResponse{
			Changes: []*traits.PullAirTemperatureResponse_Change{
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

	_, err := a.TemperatureValue.Set(&traits.AirTemperature{
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
