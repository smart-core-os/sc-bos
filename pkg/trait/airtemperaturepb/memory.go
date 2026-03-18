package airtemperaturepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

type MemoryDevice struct {
	airtemperaturepb.UnimplementedAirTemperatureApiServer
	airTemperature *resource.Value
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		airTemperature: resource.NewValue(
			resource.WithInitialValue(InitialAirTemperatureState()),
			resource.WithWritablePaths(&airtemperaturepb.AirTemperature{},
				"mode",
				// temperature_goal oneof options
				"temperature_set_point",
				"temperature_set_point_delta",
				"temperature_range",
			),
		),
	}
}

func InitialAirTemperatureState() *airtemperaturepb.AirTemperature {
	return &airtemperaturepb.AirTemperature{
		AmbientTemperature: &types.Temperature{ValueCelsius: 22},
		TemperatureGoal: &airtemperaturepb.AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: 22},
		},
	}
}

func (t *MemoryDevice) Register(server grpc.ServiceRegistrar) {
	airtemperaturepb.RegisterAirTemperatureApiServer(server, t)
}

func (t *MemoryDevice) GetAirTemperature(_ context.Context, req *airtemperaturepb.GetAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	return t.airTemperature.Get(resource.WithReadMask(req.ReadMask)).(*airtemperaturepb.AirTemperature), nil
}

func (t *MemoryDevice) UpdateAirTemperature(_ context.Context, request *airtemperaturepb.UpdateAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	update, err := t.airTemperature.Set(request.State, resource.WithUpdateMask(request.UpdateMask))
	return update.(*airtemperaturepb.AirTemperature), err
}

func (t *MemoryDevice) PullAirTemperature(request *airtemperaturepb.PullAirTemperatureRequest, server airtemperaturepb.AirTemperatureApi_PullAirTemperatureServer) error {
	for event := range t.airTemperature.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &airtemperaturepb.PullAirTemperatureResponse_Change{
			Name:           request.Name,
			AirTemperature: event.Value.(*airtemperaturepb.AirTemperature),
			ChangeTime:     timestamppb.New(event.ChangeTime),
		}
		err := server.Send(&airtemperaturepb.PullAirTemperatureResponse{
			Changes: []*airtemperaturepb.PullAirTemperatureResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
