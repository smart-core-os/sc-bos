package soundsensorpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	soundsensorpb.UnimplementedSoundSensorApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	soundsensorpb.RegisterSoundSensorApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetSoundLevel(_ context.Context, request *soundsensorpb.GetSoundLevelRequest) (*soundsensorpb.SoundLevel, error) {
	return m.model.GetSoundLevel(resource.WithReadMask(request.ReadMask))
}

func (m *ModelServer) PullSoundLevel(request *soundsensorpb.PullSoundLevelRequest, server grpc.ServerStreamingServer[soundsensorpb.PullSoundLevelResponse]) error {
	for change := range m.model.PullSoundLevel(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		msg := &soundsensorpb.PullSoundLevelResponse{Changes: []*soundsensorpb.PullSoundLevelResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(change.ChangeTime),
			SoundLevel: change.Value,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
