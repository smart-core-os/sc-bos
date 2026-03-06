package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceutilisationpb"
)

type ResourceUtilisationServer struct {
	resourceutilisationpb.UnimplementedResourceUtilisationHistoryServer
	store history.Store // payloads of *resourceutilisationpb.ResourceUtilisation
}

func NewResourceUtilisationServer(store history.Store) *ResourceUtilisationServer {
	return &ResourceUtilisationServer{store: store}
}

func (s *ResourceUtilisationServer) Register(server *grpc.Server) {
	resourceutilisationpb.RegisterResourceUtilisationHistoryServer(server, s)
}

func (s *ResourceUtilisationServer) Unwrap() any {
	return s.store
}

var resourceUtilisationPager = NewPageReader(func(r history.Record) (*resourceutilisationpb.ResourceUtilisationRecord, error) {
	v := &resourceutilisationpb.ResourceUtilisation{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &resourceutilisationpb.ResourceUtilisationRecord{
		RecordTime:          timestamppb.New(r.CreateTime),
		ResourceUtilisation: v,
	}, nil
})

func (s *ResourceUtilisationServer) ListResourceUtilisationHistory(ctx context.Context, req *resourceutilisationpb.ListResourceUtilisationHistoryRequest) (*resourceutilisationpb.ListResourceUtilisationHistoryResponse, error) {
	page, size, nextToken, err := resourceUtilisationPager.ListRecords(ctx, s.store, req.Period, int(req.PageSize), req.PageToken, req.OrderBy)
	if err != nil {
		return nil, err
	}
	return &resourceutilisationpb.ListResourceUtilisationHistoryResponse{
		TotalSize:                    int32(size),
		NextPageToken:                nextToken,
		ResourceUtilisationRecords:   page,
	}, nil
}
