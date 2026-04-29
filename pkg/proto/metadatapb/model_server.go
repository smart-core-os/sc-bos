package metadatapb

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedMetadataApiServer

	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	s := &ModelServer{model: model}
	return s
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterMetadataApiServer(server, s)
}

func (s *ModelServer) GetMetadata(_ context.Context, request *GetMetadataRequest) (*Metadata, error) {
	return s.model.GetMetadata(resource.WithReadMask(request.ReadMask))
}

func (s *ModelServer) PullMetadata(request *PullMetadataRequest, server MetadataApi_PullMetadataServer) error {
	for change := range s.model.PullMetadata(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullMetadataResponse{Changes: []*PullMetadataResponse_Change{
			{Name: request.Name, ChangeTime: change.ChangeTime, Metadata: change.Metadata},
		}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err() // the loop only ends when the context is done
}
