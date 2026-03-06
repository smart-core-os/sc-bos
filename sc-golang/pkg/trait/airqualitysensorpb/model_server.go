package airqualitysensorpb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/resource"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
)

type ModelServer struct {
	traits.UnimplementedAirQualitySensorApiServer
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
	traits.RegisterAirQualitySensorApiServer(server, s)
}

func (s *ModelServer) GetAirQuality(ctx context.Context, req *traits.GetAirQualityRequest) (*traits.AirQuality, error) {
	return s.model.GetAirQuality(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) PullAirQuality(request *traits.PullAirQualityRequest, server traits.AirQualitySensorApi_PullAirQualityServer) error {
	for update := range s.model.PullAirQuality(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullAirQualityResponse_Change{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			AirQuality: update.Value,
		}

		err := server.Send(&traits.PullAirQualityResponse{
			Changes: []*traits.PullAirQualityResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
