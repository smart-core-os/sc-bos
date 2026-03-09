package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/rebootpb"
)

type RebootServer struct {
	rebootpb.UnimplementedRebootHistoryServer
	store history.Store // payloads of *rebootpb.RebootState
}

func NewRebootServer(store history.Store) *RebootServer {
	return &RebootServer{store: store}
}

func (s *RebootServer) Register(server *grpc.Server) {
	rebootpb.RegisterRebootHistoryServer(server, s)
}

func (s *RebootServer) Unwrap() any {
	return s.store
}

var rebootPager = NewPageReader(func(r history.Record) (*rebootpb.RebootEvent, error) {
	v := &rebootpb.RebootState{}
	if err := proto.Unmarshal(r.Payload, v); err != nil {
		return nil, err
	}
	event := &rebootpb.RebootEvent{
		RebootTime: v.BootTime,
		Reason:     v.LastRebootReason,
	}
	if v.LastRebootActor != "" {
		event.Actor = &actorpb.Actor{DisplayName: v.LastRebootActor}
	}
	return event, nil
})

func (s *RebootServer) ListRebootEvents(ctx context.Context, req *rebootpb.ListRebootEventsRequest) (*rebootpb.ListRebootEventsResponse, error) {
	page, size, nextToken, err := rebootPager.ListRecords(ctx, s.store, req.Period, int(req.PageSize), req.PageToken, req.OrderBy)
	if err != nil {
		return nil, err
	}
	return &rebootpb.ListRebootEventsResponse{
		TotalSize:     int32(size),
		NextPageToken: nextToken,
		RebootEvents:  page,
	}, nil
}
