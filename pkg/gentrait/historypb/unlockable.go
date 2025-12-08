package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/history"
)

type UnlockableServer struct {
	gen.UnimplementedUnlockableAPIServer
	store history.Store // payloads of *gen.UnlockableRecord
}

func NewUnlockableServer(store history.Store) *UnlockableServer {
	return &UnlockableServer{store: store}
}

func (l *UnlockableServer) Register(server *grpc.Server) {
	gen.RegisterUnlockableAPIServer(server, l)
}

func (l *UnlockableServer) Unwrap() any {
	return l.store
}

var unlockableRecordPager = NewPageReader(func(r history.Record) (*gen.UnlockableRecord, error) {
	v := &gen.Unlockable{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &gen.UnlockableRecord{
		RecordTime: timestamppb.New(r.CreateTime),
		Unlockable: v,
	}, nil
})

func (l *UnlockableServer) ListLockerHistory(ctx context.Context, request *gen.ListUnlockableHistoryRequest) (*gen.ListUnlockableHistoryResponse, error) {
	page, size, nextToken, err := unlockableRecordPager.ListRecords(ctx, l.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &gen.ListUnlockableHistoryResponse{
		TotalSize:         int32(size),
		NextPageToken:     nextToken,
		UnlockableRecords: page,
	}, nil
}
