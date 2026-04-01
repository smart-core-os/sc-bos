package countpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedCountApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterCountApiServer(server, s)
}

func (s *ModelServer) GetCount(_ context.Context, req *GetCountRequest) (*Count, error) {
	return s.model.GetCount(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateCount(_ context.Context, req *UpdateCountRequest) (*Count, error) {
	return s.model.UpdateCount(req.Count, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) ResetCount(_ context.Context, _ *ResetCountRequest) (*Count, error) {
	return s.model.UpdateCount(&Count{})
}

func (s *ModelServer) PullCounts(request *PullCountsRequest, server CountApi_PullCountsServer) error {
	for update := range s.model.PullCounts(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullCountsResponse{Changes: []*PullCountsResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			Count:      update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
