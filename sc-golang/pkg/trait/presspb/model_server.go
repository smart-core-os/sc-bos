package presspb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	traits.UnimplementedPressApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) GetButtonState(ctx context.Context, request *traits.GetPressedStateRequest) (*traits.PressedState, error) {
	return s.model.GetPressedState(resource.WithReadMask(request.ReadMask)), nil
}

func (s *ModelServer) UpdateButtonState(ctx context.Context, request *traits.UpdatePressedStateRequest) (*traits.PressedState, error) {
	return s.model.UpdatePressedState(request.PressedState, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullButtonState(request *traits.PullPressedStateRequest, server traits.PressApi_PullPressedStateServer) error {
	changes := s.model.PullPressedState(server.Context(),
		resource.WithReadMask(request.ReadMask),
		resource.WithUpdatesOnly(request.UpdatesOnly),
	)
	for change := range changes {
		err := server.Send(&traits.PullPressedStateResponse{
			Changes: []*traits.PullPressedStateResponse_Change{
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
