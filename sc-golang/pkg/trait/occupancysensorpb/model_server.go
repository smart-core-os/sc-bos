package occupancysensorpb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/resource"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
)

type ModelServer struct {
	traits.UnimplementedOccupancySensorApiServer
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
	traits.RegisterOccupancySensorApiServer(server, s)
}

func (s *ModelServer) GetOccupancy(_ context.Context, req *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	return s.model.GetOccupancy(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) PullOccupancy(request *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	for update := range s.model.PullOccupancy(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullOccupancyResponse_Change{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Occupancy:  update.Value,
		}

		err := server.Send(&traits.PullOccupancyResponse{
			Changes: []*traits.PullOccupancyResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
