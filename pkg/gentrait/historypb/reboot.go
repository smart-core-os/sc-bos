package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/bootpb"
)

type BootServer struct {
	bootpb.UnimplementedBootHistoryServer
	store history.Store // payloads of *bootpb.BootState
}

func NewBootServer(store history.Store) *BootServer {
	return &BootServer{store: store}
}

func (s *BootServer) Register(server *grpc.Server) {
	bootpb.RegisterBootHistoryServer(server, s)
}

func (s *BootServer) Unwrap() any {
	return s.store
}

var bootPager = NewPageReader(func(r history.Record) (*bootpb.BootEvent, error) {
	v := &bootpb.BootState{}
	if err := proto.Unmarshal(r.Payload, v); err != nil {
		return nil, err
	}
	return &bootpb.BootEvent{
		RebootTime: v.BootTime,
		Reason:     v.LastRebootReason,
		Actor:      v.LastRebootActor,
	}, nil
})

func (s *BootServer) ListBootEvents(ctx context.Context, req *bootpb.ListBootEventsRequest) (*bootpb.ListBootEventsResponse, error) {
	page, size, nextToken, err := bootPager.ListRecords(ctx, s.store, req.Period, int(req.PageSize), req.PageToken, req.OrderBy)
	if err != nil {
		return nil, err
	}
	return &bootpb.ListBootEventsResponse{
		TotalSize:     int32(size),
		NextPageToken: nextToken,
		BootEvents:    page,
	}, nil
}
