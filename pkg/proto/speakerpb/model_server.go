package speakerpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedSpeakerApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterSpeakerApiServer(server, s)
}

func (s *ModelServer) GetVolume(_ context.Context, req *GetSpeakerVolumeRequest) (*typespb.AudioLevel, error) {
	return s.model.GetVolume(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateVolume(_ context.Context, req *UpdateSpeakerVolumeRequest) (*typespb.AudioLevel, error) {
	return s.model.UpdateVolume(req.Volume, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) PullVolume(request *PullSpeakerVolumeRequest, server SpeakerApi_PullVolumeServer) error {
	for update := range s.model.PullVolume(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullSpeakerVolumeResponse{
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
