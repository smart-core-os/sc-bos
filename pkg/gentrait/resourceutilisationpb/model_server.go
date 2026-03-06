package resourceutilisationpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/resourceutilisationpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	resourceutilisationpb.UnimplementedResourceUtilisationApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Register(server *grpc.Server) {
	resourceutilisationpb.RegisterResourceUtilisationApiServer(server, s)
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) GetResourceUtilisation(_ context.Context, req *resourceutilisationpb.GetResourceUtilisationRequest) (*resourceutilisationpb.ResourceUtilisation, error) {
	return s.model.GetResourceUtilisation(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) PullResourceUtilisation(req *resourceutilisationpb.PullResourceUtilisationRequest, server resourceutilisationpb.ResourceUtilisationApi_PullResourceUtilisationServer) error {
	for change := range s.model.PullResourceUtilisation(server.Context(), resource.WithReadMask(req.ReadMask), resource.WithUpdatesOnly(req.UpdatesOnly)) {
		msg := &resourceutilisationpb.PullResourceUtilisationResponse{
			Changes: []*resourceutilisationpb.PullResourceUtilisationResponse_Change{{
				Name:                 req.Name,
				ChangeTime:           timestamppb.New(change.ChangeTime),
				ResourceUtilisation:  change.Value,
			}},
		}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
