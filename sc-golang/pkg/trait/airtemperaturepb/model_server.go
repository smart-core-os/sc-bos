package airtemperaturepb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/resource"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
)

type ModelServer struct {
	traits.UnimplementedAirTemperatureApiServer
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
	traits.RegisterAirTemperatureApiServer(server, s)
}

func (s *ModelServer) GetAirTemperature(_ context.Context, req *traits.GetAirTemperatureRequest) (*traits.AirTemperature, error) {
	return s.model.GetAirTemperature(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateAirTemperature(_ context.Context, request *traits.UpdateAirTemperatureRequest) (*traits.AirTemperature, error) {
	return s.model.UpdateAirTemperature(request.State, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullAirTemperature(request *traits.PullAirTemperatureRequest, server traits.AirTemperatureApi_PullAirTemperatureServer) error {
	for update := range s.model.PullAirTemperature(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullAirTemperatureResponse_Change{
			Name:           request.Name,
			ChangeTime:     timestamppb.New(update.ChangeTime),
			AirTemperature: update.Value,
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
