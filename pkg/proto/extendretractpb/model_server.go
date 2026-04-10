package extendretractpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedExtendRetractApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterExtendRetractApiServer(server, s)
}

func (s *ModelServer) GetExtension(_ context.Context, req *GetExtensionRequest) (*Extension, error) {
	return s.model.GetExtension(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateExtension(_ context.Context, req *UpdateExtensionRequest) (*Extension, error) {
	return s.model.UpdateExtension(req.Extension, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) Stop(_ context.Context, _ *ExtendRetractStopRequest) (*Extension, error) {
	return s.model.GetExtension()
}

func (s *ModelServer) CreateExtensionPreset(_ context.Context, _ *CreateExtensionPresetRequest) (*ExtensionPreset, error) {
	return nil, status.Errorf(codes.Unimplemented, "CreateExtensionPreset not implemented")
}

func (s *ModelServer) PullExtensions(request *PullExtensionsRequest, server ExtendRetractApi_PullExtensionsServer) error {
	for update := range s.model.PullExtensions(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullExtensionsResponse{Changes: []*PullExtensionsResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Extension:  update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
