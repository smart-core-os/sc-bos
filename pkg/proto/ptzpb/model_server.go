package ptzpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedPtzApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterPtzApiServer(server, s)
}

func (s *ModelServer) GetPtz(_ context.Context, req *GetPtzRequest) (*Ptz, error) {
	return s.model.GetPtz(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdatePtz(_ context.Context, req *UpdatePtzRequest) (*Ptz, error) {
	return s.model.UpdatePtz(req.State, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) Stop(_ context.Context, _ *StopPtzRequest) (*Ptz, error) {
	return s.model.GetPtz()
}

func (s *ModelServer) CreatePreset(_ context.Context, _ *CreatePtzPresetRequest) (*PtzPreset, error) {
	return nil, status.Errorf(codes.Unimplemented, "CreatePreset not implemented")
}

func (s *ModelServer) PullPtz(request *PullPtzRequest, server PtzApi_PullPtzServer) error {
	for update := range s.model.PullPtz(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullPtzResponse{Changes: []*PullPtzResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Ptz:        update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
