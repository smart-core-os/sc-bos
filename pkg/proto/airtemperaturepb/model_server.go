package airtemperaturepb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"

	"google.golang.org/grpc"
)

type ModelServer struct {
	UnimplementedAirTemperatureApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterAirTemperatureApiServer(server, s)
}

func (s *ModelServer) GetAirTemperature(_ context.Context, req *GetAirTemperatureRequest) (*AirTemperature, error) {
	return s.model.GetAirTemperature(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateAirTemperature(_ context.Context, request *UpdateAirTemperatureRequest) (*AirTemperature, error) {
	return s.model.UpdateAirTemperature(request.State, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullAirTemperature(request *PullAirTemperatureRequest, server AirTemperatureApi_PullAirTemperatureServer) error {
	for update := range s.model.PullAirTemperature(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullAirTemperatureResponse_Change{
			Name:           request.Name,
			ChangeTime:     timestamppb.New(update.ChangeTime),
			AirTemperature: update.Value,
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
