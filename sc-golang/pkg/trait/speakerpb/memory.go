package speakerpb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/resource"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
)

type MemoryDevice struct {
	traits.UnimplementedSpeakerApiServer
	volume *resource.Value
}

func NewMemoryDevice(initialState *types.AudioLevel) *MemoryDevice {
	return &MemoryDevice{
		volume: resource.NewValue(resource.WithInitialValue(initialState)),
	}
}

func (s *MemoryDevice) Register(server grpc.ServiceRegistrar) {
	traits.RegisterSpeakerApiServer(server, s)
}

func (s *MemoryDevice) GetVolume(_ context.Context, req *traits.GetSpeakerVolumeRequest) (*types.AudioLevel, error) {
	return s.volume.Get(resource.WithReadMask(req.ReadMask)).(*types.AudioLevel), nil
}

func (s *MemoryDevice) UpdateVolume(_ context.Context, request *traits.UpdateSpeakerVolumeRequest) (*types.AudioLevel, error) {
	newValue, err := s.volume.Set(request.Volume, resource.WithUpdateMask(request.UpdateMask), resource.InterceptBefore(func(old, change proto.Message) {
		if request.Delta {
			val := old.(*types.AudioLevel)
			delta := change.(*types.AudioLevel)
			delta.Gain += val.Gain
		}
	}))
	if err != nil {
		return nil, err
	}
	return newValue.(*types.AudioLevel), nil
}

func (s *MemoryDevice) PullVolume(request *traits.PullSpeakerVolumeRequest, server traits.SpeakerApi_PullVolumeServer) error {
	for change := range s.volume.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		typedChange := &types.AudioLevelChange{
			Name:       request.Name,
			Level:      change.Value.(*types.AudioLevel),
			ChangeTime: timestamppb.New(change.ChangeTime),
		}
		err := server.Send(&traits.PullSpeakerVolumeResponse{
			Changes: []*types.AudioLevelChange{typedChange},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}
