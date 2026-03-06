package resourceusepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	resourceusepb.UnimplementedResourceUseApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Register(server *grpc.Server) {
	resourceusepb.RegisterResourceUseApiServer(server, s)
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) GetResourceUse(_ context.Context, req *resourceusepb.GetResourceUseRequest) (*resourceusepb.ResourceUse, error) {
	return s.model.GetResourceUse(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) PullResourceUse(req *resourceusepb.PullResourceUseRequest, server resourceusepb.ResourceUseApi_PullResourceUseServer) error {
	for change := range s.model.PullResourceUse(server.Context(), resource.WithReadMask(req.ReadMask), resource.WithUpdatesOnly(req.UpdatesOnly)) {
		msg := &resourceusepb.PullResourceUseResponse{
			Changes: []*resourceusepb.PullResourceUseResponse_Change{{
				Name:        req.Name,
				ChangeTime:  timestamppb.New(change.ChangeTime),
				ResourceUse: change.Value,
			}},
		}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
