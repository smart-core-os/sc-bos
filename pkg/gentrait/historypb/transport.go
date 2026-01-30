package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
)

type TransportServer struct {
	transportpb.UnimplementedTransportHistoryServer
	store history.Store // payloads of *transportpb.Transport
}

func NewTransportServer(store history.Store) *TransportServer {
	return &TransportServer{store: store}
}

func (m *TransportServer) Register(server *grpc.Server) {
	transportpb.RegisterTransportHistoryServer(server, m)
}

func (m *TransportServer) Unwrap() any {
	return m.store
}

var transportPager = NewPageReader(func(r history.Record) (*transportpb.TransportRecord, error) {
	v := &transportpb.Transport{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &transportpb.TransportRecord{
		RecordTime: timestamppb.New(r.CreateTime),
		Transport:  v,
	}, nil
})

func (m *TransportServer) ListTransportHistory(ctx context.Context, request *transportpb.ListTransportHistoryRequest) (*transportpb.ListTransportHistoryResponse, error) {
	page, size, nextToken, err := transportPager.ListRecords(ctx, m.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &transportpb.ListTransportHistoryResponse{
		TotalSize:        int32(size),
		NextPageToken:    nextToken,
		TransportRecords: page,
	}, nil
}
