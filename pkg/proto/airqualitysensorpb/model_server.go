package airqualitysensorpb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"

	"google.golang.org/grpc"
)

type ModelServer struct {
	UnimplementedAirQualitySensorApiServer
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
	RegisterAirQualitySensorApiServer(server, s)
}

func (s *ModelServer) GetAirQuality(ctx context.Context, req *GetAirQualityRequest) (*AirQuality, error) {
	return s.model.GetAirQuality(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) PullAirQuality(request *PullAirQualityRequest, server AirQualitySensorApi_PullAirQualityServer) error {
	for update := range s.model.PullAirQuality(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullAirQualityResponse_Change{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			AirQuality: update.Value,
		}

		err := server.Send(&PullAirQualityResponse{
			Changes: []*PullAirQualityResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
