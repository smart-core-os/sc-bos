package airtemperaturepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type MemoryDevice struct {
	traits.UnimplementedAirTemperatureApiServer
	airTemperature *resource.Value
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		airTemperature: resource.NewValue(
			resource.WithInitialValue(InitialAirTemperatureState()),
			resource.WithWritablePaths(&traits.AirTemperature{},
				"mode",
				// temperature_goal oneof options
				"temperature_set_point",
				"temperature_set_point_delta",
				"temperature_range",
			),
		),
	}
}

func InitialAirTemperatureState() *traits.AirTemperature {
	return &traits.AirTemperature{
		AmbientTemperature: &types.Temperature{ValueCelsius: 22},
		TemperatureGoal: &traits.AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: 22},
		},
	}
}

func (t *MemoryDevice) Register(server grpc.ServiceRegistrar) {
	traits.RegisterAirTemperatureApiServer(server, t)
}

func (t *MemoryDevice) GetAirTemperature(_ context.Context, req *traits.GetAirTemperatureRequest) (*traits.AirTemperature, error) {
	return t.airTemperature.Get(resource.WithReadMask(req.ReadMask)).(*traits.AirTemperature), nil
}

func (t *MemoryDevice) UpdateAirTemperature(_ context.Context, request *traits.UpdateAirTemperatureRequest) (*traits.AirTemperature, error) {
	update, err := t.airTemperature.Set(request.State, resource.WithUpdateMask(request.UpdateMask))
	return update.(*traits.AirTemperature), err
}

func (t *MemoryDevice) PullAirTemperature(request *traits.PullAirTemperatureRequest, server traits.AirTemperatureApi_PullAirTemperatureServer) error {
	for event := range t.airTemperature.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullAirTemperatureResponse_Change{
			Name:           request.Name,
			AirTemperature: event.Value.(*traits.AirTemperature),
			ChangeTime:     timestamppb.New(event.ChangeTime),
		}
		err := server.Send(&traits.PullAirTemperatureResponse{
			Changes: []*traits.PullAirTemperatureResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
