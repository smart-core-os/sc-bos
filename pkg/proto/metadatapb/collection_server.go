package metadatapb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type CollectionServer struct {
	UnimplementedMetadataApiServer
	model *Collection
}

func NewCollectionServer(model *Collection) *CollectionServer {
	s := &CollectionServer{model: model}
	return s
}

func (s *CollectionServer) Unwrap() any {
	return s.model
}

func (s *CollectionServer) GetMetadata(_ context.Context, request *GetMetadataRequest) (*Metadata, error) {
	return s.model.GetMetadata(request.Name)
}

func (s *CollectionServer) PullMetadata(request *PullMetadataRequest, server MetadataApi_PullMetadataServer) error {
	for change := range s.model.PullMetadata(server.Context(), request.Name, resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullMetadataResponse{Changes: []*PullMetadataResponse_Change{
			{Name: request.Name, ChangeTime: change.ChangeTime, Metadata: change.Metadata},
		}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err() // the loop only ends when the context is done
}
