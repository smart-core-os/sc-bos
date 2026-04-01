package inputselectpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedInputSelectApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterInputSelectApiServer(server, s)
}

func (s *ModelServer) GetInput(_ context.Context, req *GetInputRequest) (*Input, error) {
	return s.model.GetInput(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateInput(_ context.Context, req *UpdateInputRequest) (*Input, error) {
	return s.model.UpdateInput(req.Input, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) PullInput(request *PullInputRequest, server InputSelectApi_PullInputServer) error {
	for update := range s.model.PullInput(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullInputResponse{Changes: []*PullInputResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Input:      update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
