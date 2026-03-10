package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
)

type ResourceUseServer struct {
	resourceusepb.UnimplementedResourceUseHistoryServer
	store history.Store // payloads of *resourceusepb.ResourceUse
}

func NewResourceUseServer(store history.Store) *ResourceUseServer {
	return &ResourceUseServer{store: store}
}

func (s *ResourceUseServer) Register(server *grpc.Server) {
	resourceusepb.RegisterResourceUseHistoryServer(server, s)
}

func (s *ResourceUseServer) Unwrap() any {
	return s.store
}

var resourceUsePager = NewPageReader(func(r history.Record) (*resourceusepb.ResourceUseRecord, error) {
	v := &resourceusepb.ResourceUse{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &resourceusepb.ResourceUseRecord{
		RecordTime:  timestamppb.New(r.CreateTime),
		ResourceUse: v,
	}, nil
})

func (s *ResourceUseServer) ListResourceUseHistory(ctx context.Context, req *resourceusepb.ListResourceUseHistoryRequest) (*resourceusepb.ListResourceUseHistoryResponse, error) {
	page, size, nextToken, err := resourceUsePager.ListRecords(ctx, s.store, req.Period, int(req.PageSize), req.PageToken, req.OrderBy)
	if err != nil {
		return nil, err
	}
	return &resourceusepb.ListResourceUseHistoryResponse{
		TotalSize:          int32(size),
		NextPageToken:      nextToken,
		ResourceUseRecords: page,
	}, nil
}
