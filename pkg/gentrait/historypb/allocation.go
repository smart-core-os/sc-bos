package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
)

type AllocationServer struct {
	allocationpb.UnimplementedAllocationHistoryServer
	store history.Store // payloads of *allocationpb.Allocation
}

var allocationPager = NewPageReader(func(r history.Record) (*allocationpb.AllocationRecord, error) {
	v := &allocationpb.Allocation{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &allocationpb.AllocationRecord{
		RecordTime: timestamppb.New(r.CreateTime),
		Allocation: v,
	}, nil
})

func (a *AllocationServer) ListAllocationHistory(ctx context.Context, request *allocationpb.ListAllocationHistoryRequest) (*allocationpb.ListAllocationHistoryResponse, error) {
	page, size, nextToken, err := allocationPager.ListRecords(ctx, a.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &allocationpb.ListAllocationHistoryResponse{
		TotalSize:         int32(size),
		NextPageToken:     nextToken,
		AllocationRecords: page,
	}, nil
}

func NewAllocationServer(store history.Store) *AllocationServer {
	return &AllocationServer{store: store}
}

func (a *AllocationServer) Register(server *grpc.Server) {
	allocationpb.RegisterAllocationHistoryServer(server, a)
}

func (a *AllocationServer) Unwrap() any {
	return a.store
}
