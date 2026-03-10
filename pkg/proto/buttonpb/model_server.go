package buttonpb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	UnimplementedButtonApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) GetButtonState(ctx context.Context, request *GetButtonStateRequest) (*ButtonState, error) {
	return s.model.GetButtonState(resource.WithReadMask(request.ReadMask)), nil
}

func (s *ModelServer) UpdateButtonState(ctx context.Context, request *UpdateButtonStateRequest) (*ButtonState, error) {
	return s.model.UpdateButtonState(request.ButtonState, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullButtonState(request *PullButtonStateRequest, server ButtonApi_PullButtonStateServer) error {
	changes := s.model.PullButtonState(server.Context(),
		resource.WithReadMask(request.ReadMask),
		resource.WithUpdatesOnly(request.UpdatesOnly),
	)
	for change := range changes {
		err := server.Send(&PullButtonStateResponse{
			Changes: []*PullButtonStateResponse_Change{
				{
					Name:        request.Name,
					ChangeTime:  timestamppb.New(change.ChangeTime),
					ButtonState: change.Value,
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
