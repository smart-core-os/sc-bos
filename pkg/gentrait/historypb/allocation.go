package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/history"
)

type AllocationServer struct {
	gen.UnimplementedAllocationHistoryServer
	store history.Store // payloads of *gen.Allocation
}

var allocationPager = NewPageReader(func(r history.Record) (*gen.AllocationRecord, error) {
	v := &gen.Allocation{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &gen.AllocationRecord{
		RecordTime: timestamppb.New(r.CreateTime),
		Allocation: v,
	}, nil
})

func (a *AllocationServer) ListAllocationHistory(ctx context.Context, request *gen.ListAllocationHistoryRequest) (*gen.ListAllocationHistoryResponse, error) {
	page, size, nextToken, err := allocationPager.ListRecords(ctx, a.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &gen.ListAllocationHistoryResponse{
		TotalSize:         int32(size),
		NextPageToken:     nextToken,
		AllocationRecords: page,
	}, nil
}

func NewAllocationServer(store history.Store) *AllocationServer {
	return &AllocationServer{store: store}
}

func (a *AllocationServer) Register(server *grpc.Server) {
	gen.RegisterAllocationHistoryServer(server, a)
}

func (a *AllocationServer) Unwrap() any {
	return a.store
}
