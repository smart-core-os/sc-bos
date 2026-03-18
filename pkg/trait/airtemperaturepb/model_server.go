package airtemperaturepb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/resource"

	"google.golang.org/grpc"
)

type ModelServer struct {
	airtemperaturepb.UnimplementedAirTemperatureApiServer
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
	airtemperaturepb.RegisterAirTemperatureApiServer(server, s)
}

func (s *ModelServer) GetAirTemperature(_ context.Context, req *airtemperaturepb.GetAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	return s.model.GetAirTemperature(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateAirTemperature(_ context.Context, request *airtemperaturepb.UpdateAirTemperatureRequest) (*airtemperaturepb.AirTemperature, error) {
	return s.model.UpdateAirTemperature(request.State, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullAirTemperature(request *airtemperaturepb.PullAirTemperatureRequest, server airtemperaturepb.AirTemperatureApi_PullAirTemperatureServer) error {
	for update := range s.model.PullAirTemperature(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &airtemperaturepb.PullAirTemperatureResponse_Change{
			Name:           request.Name,
			ChangeTime:     timestamppb.New(update.ChangeTime),
			AirTemperature: update.Value,
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
