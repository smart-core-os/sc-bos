package presspb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedPressApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) GetButtonState(ctx context.Context, request *GetPressedStateRequest) (*PressedState, error) {
	return s.model.GetPressedState(resource.WithReadMask(request.ReadMask)), nil
}

func (s *ModelServer) UpdateButtonState(ctx context.Context, request *UpdatePressedStateRequest) (*PressedState, error) {
	return s.model.UpdatePressedState(request.PressedState, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullButtonState(request *PullPressedStateRequest, server PressApi_PullPressedStateServer) error {
	changes := s.model.PullPressedState(server.Context(),
		resource.WithReadMask(request.ReadMask),
		resource.WithUpdatesOnly(request.UpdatesOnly),
	)
	for change := range changes {
		err := server.Send(&PullPressedStateResponse{
			Changes: []*PullPressedStateResponse_Change{
				{
					Name:         request.Name,
					ChangeTime:   timestamppb.New(change.ChangeTime),
					PressedState: change.Value,
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
