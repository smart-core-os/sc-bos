package brightnesssensorpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	traits.UnimplementedBrightnessSensorApiServer

	model *Model
}

func NewModelServer(model *Model, opts ...ServerOption) *ModelServer {
	s := &ModelServer{model: model}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	traits.RegisterBrightnessSensorApiServer(server, s)
}

func (s *ModelServer) GetAmbientBrightness(_ context.Context, request *traits.GetAmbientBrightnessRequest) (*traits.AmbientBrightness, error) {
	return s.model.GetAmbientBrightness(resource.WithReadMask(request.GetReadMask()))
}

func (s *ModelServer) PullAmbientBrightness(request *traits.PullAmbientBrightnessRequest, server traits.BrightnessSensorApi_PullAmbientBrightnessServer) error {
	for update := range s.model.PullAmbientBrightness(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullAmbientBrightnessResponse_Change{
			Name:              request.Name,
			ChangeTime:        timestamppb.New(update.ChangeTime),
			AmbientBrightness: update.Value,
		}

		err := server.Send(&traits.PullAmbientBrightnessResponse{
			Changes: []*traits.PullAmbientBrightnessResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}

type ServerOption func(s *ModelServer)
