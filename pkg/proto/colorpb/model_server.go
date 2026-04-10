package colorpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedColorApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterColorApiServer(server, s)
}

func (s *ModelServer) GetColor(_ context.Context, req *GetColorRequest) (*Color, error) {
	return s.model.GetColor(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateColor(_ context.Context, req *UpdateColorRequest) (*Color, error) {
	return s.model.UpdateColor(req.Color, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) PullColor(request *PullColorRequest, server ColorApi_PullColorServer) error {
	for update := range s.model.PullColor(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullColorResponse{Changes: []*PullColorResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Color:      update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
