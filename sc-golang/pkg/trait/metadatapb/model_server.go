package metadatapb

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	traits.UnimplementedMetadataApiServer

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
	traits.RegisterMetadataApiServer(server, s)
}

func (s *ModelServer) GetMetadata(_ context.Context, request *traits.GetMetadataRequest) (*traits.Metadata, error) {
	return s.model.GetMetadata(resource.WithReadMask(request.ReadMask))
}

func (s *ModelServer) PullMetadata(request *traits.PullMetadataRequest, server traits.MetadataApi_PullMetadataServer) error {
	for change := range s.model.PullMetadata(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullMetadataResponse{Changes: []*traits.PullMetadataResponse_Change{
			{Name: request.Name, ChangeTime: change.ChangeTime, Metadata: change.Metadata},
		}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err() // the loop only ends when the context is done
}
