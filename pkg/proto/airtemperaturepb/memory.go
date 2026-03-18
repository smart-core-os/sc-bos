package airtemperaturepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

type MemoryDevice struct {
	UnimplementedAirTemperatureApiServer
	airTemperature *resource.Value
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		airTemperature: resource.NewValue(
			resource.WithInitialValue(InitialAirTemperatureState()),
			resource.WithWritablePaths(&AirTemperature{},
				"mode",
				// temperature_goal oneof options
				"temperature_set_point",
				"temperature_set_point_delta",
				"temperature_range",
			),
		),
	}
}

func InitialAirTemperatureState() *AirTemperature {
	return &AirTemperature{
		AmbientTemperature: &types.Temperature{ValueCelsius: 22},
		TemperatureGoal: &AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: 22},
		},
	}
}

func (t *MemoryDevice) Register(server grpc.ServiceRegistrar) {
	RegisterAirTemperatureApiServer(server, t)
}

func (t *MemoryDevice) GetAirTemperature(_ context.Context, req *GetAirTemperatureRequest) (*AirTemperature, error) {
	return t.airTemperature.Get(resource.WithReadMask(req.ReadMask)).(*AirTemperature), nil
}

func (t *MemoryDevice) UpdateAirTemperature(_ context.Context, request *UpdateAirTemperatureRequest) (*AirTemperature, error) {
	update, err := t.airTemperature.Set(request.State, resource.WithUpdateMask(request.UpdateMask))
	return update.(*AirTemperature), err
}

func (t *MemoryDevice) PullAirTemperature(request *PullAirTemperatureRequest, server AirTemperatureApi_PullAirTemperatureServer) error {
	for event := range t.airTemperature.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullAirTemperatureResponse_Change{
			Name:           request.Name,
			AirTemperature: event.Value.(*AirTemperature),
			ChangeTime:     timestamppb.New(event.ChangeTime),
		}
		err := server.Send(&PullAirTemperatureResponse{
			Changes: []*PullAirTemperatureResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
