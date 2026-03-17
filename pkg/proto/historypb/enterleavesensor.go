package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
)

type EnterLeaveSensorServer struct {
	enterleavesensorpb.UnimplementedEnterLeaveSensorHistoryServer
	store history.Store // payloads of *traits.EnterLeaveEvent
}

func NewEnterLeaveSensorServer(store history.Store) *EnterLeaveSensorServer {
	return &EnterLeaveSensorServer{store: store}
}

func (e *EnterLeaveSensorServer) Register(server *grpc.Server) {
	enterleavesensorpb.RegisterEnterLeaveSensorHistoryServer(server, e)
}

func (e *EnterLeaveSensorServer) Unwrap() any {
	return e.store
}

var enterLeaveEventPager = NewPageReader(func(r history.Record) (*enterleavesensorpb.EnterLeaveEventRecord, error) {
	v := &traits.EnterLeaveEvent{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &enterleavesensorpb.EnterLeaveEventRecord{
		RecordTime:      timestamppb.New(r.CreateTime),
		EnterLeaveEvent: v,
	}, nil
})

func (e *EnterLeaveSensorServer) ListEnterLeaveSensorHistory(ctx context.Context, request *enterleavesensorpb.ListEnterLeaveHistoryRequest) (*enterleavesensorpb.ListEnterLeaveHistoryResponse, error) {
	page, size, nextToken, err := enterLeaveEventPager.ListRecords(ctx, e.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &enterleavesensorpb.ListEnterLeaveHistoryResponse{
		TotalSize:         int32(size),
		NextPageToken:     nextToken,
		EnterLeaveRecords: page,
	}, nil
}
