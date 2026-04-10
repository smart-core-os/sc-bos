package microphonepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedMicrophoneApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterMicrophoneApiServer(server, s)
}

func (s *ModelServer) GetGain(_ context.Context, req *GetMicrophoneGainRequest) (*typespb.AudioLevel, error) {
	return s.model.GetGain(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateGain(_ context.Context, req *UpdateMicrophoneGainRequest) (*typespb.AudioLevel, error) {
	return s.model.UpdateGain(req.Gain, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) PullGain(request *PullMicrophoneGainRequest, server MicrophoneApi_PullGainServer) error {
	for update := range s.model.PullGain(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullMicrophoneGainResponse{
			Changes: []*typespb.AudioLevelChange{{
				Name:       request.Name,
				ChangeTime: timestamppb.New(update.ChangeTime),
				Level:      update.Value,
			}},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
