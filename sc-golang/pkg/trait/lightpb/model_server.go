package lightpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	traits.UnimplementedLightApiServer
	traits.UnimplementedLightInfoServer
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
	traits.RegisterLightApiServer(server, s)
	traits.RegisterLightInfoServer(server, s)
}

func (s *ModelServer) GetBrightness(_ context.Context, req *traits.GetBrightnessRequest) (*traits.Brightness, error) {
	return s.model.GetBrightness(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateBrightness(_ context.Context, request *traits.UpdateBrightnessRequest) (*traits.Brightness, error) {
	return s.model.UpdateBrightness(request.Brightness, resource.WithUpdateMask(request.UpdateMask))
}

func (s *ModelServer) PullBrightness(request *traits.PullBrightnessRequest, server traits.LightApi_PullBrightnessServer) error {
	for update := range s.model.PullBrightness(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		change := &traits.PullBrightnessResponse_Change{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Brightness: update.Value,
		}

		err := server.Send(&traits.PullBrightnessResponse{
			Changes: []*traits.PullBrightnessResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}

func (s *ModelServer) DescribeBrightness(_ context.Context, _ *traits.DescribeBrightnessRequest) (*traits.BrightnessSupport, error) {
	support := &traits.BrightnessSupport{
		ResourceSupport: &types.ResourceSupport{
			Readable: true, Writable: true, Observable: true,
			PullSupport: types.PullSupport_PULL_SUPPORT_NATIVE,
		},
		Presets: s.model.ListPresets(),
	}
	return support, nil
}
