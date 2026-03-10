package onoffpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	traits.UnimplementedOnOffApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	traits.RegisterOnOffApiServer(server, s)
}

func (s *ModelServer) GetOnOff(_ context.Context, req *traits.GetOnOffRequest) (*traits.OnOff, error) {
	return s.model.GetOnOff(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateOnOff(_ context.Context, request *traits.UpdateOnOffRequest) (*traits.OnOff, error) {
	return s.model.UpdateOnOff(request.OnOff, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullOnOff(request *traits.PullOnOffRequest, server traits.OnOffApi_PullOnOffServer) error {
	for update := range s.model.PullOnOff(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullOnOffResponse_Change{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			OnOff:      update.Value,
		}

		err := server.Send(&traits.PullOnOffResponse{
			Changes: []*traits.PullOnOffResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
