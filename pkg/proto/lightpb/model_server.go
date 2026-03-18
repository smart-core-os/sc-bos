package lightpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

type ModelServer struct {
	UnimplementedLightApiServer
	UnimplementedLightInfoServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterLightApiServer(server, s)
	RegisterLightInfoServer(server, s)
}

func (s *ModelServer) GetBrightness(_ context.Context, req *GetBrightnessRequest) (*Brightness, error) {
	return s.model.GetBrightness(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateBrightness(_ context.Context, request *UpdateBrightnessRequest) (*Brightness, error) {
	return s.model.UpdateBrightness(request.Brightness, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullBrightness(request *PullBrightnessRequest, server LightApi_PullBrightnessServer) error {
	for update := range s.model.PullBrightness(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &PullBrightnessResponse_Change{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Brightness: update.Value,
		}

		err := server.Send(&PullBrightnessResponse{
			Changes: []*PullBrightnessResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}

func (s *ModelServer) DescribeBrightness(_ context.Context, _ *DescribeBrightnessRequest) (*BrightnessSupport, error) {
	support := &BrightnessSupport{
		ResourceSupport: &types.ResourceSupport{
			Readable: true, Writable: true, Observable: true,
			PullSupport: types.PullSupport_PULL_SUPPORT_NATIVE,
		},
		Presets: s.model.ListPresets(),
	}
	return support, nil
}
