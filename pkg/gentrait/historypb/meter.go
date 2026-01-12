package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

type MeterServer struct {
	meterpb.UnimplementedMeterHistoryServer
	store history.Store // payloads of *meterpb.MeterReading
}

func NewMeterServer(store history.Store) *MeterServer {
	return &MeterServer{store: store}
}

func (m *MeterServer) Register(server *grpc.Server) {
	meterpb.RegisterMeterHistoryServer(server, m)
}

func (m *MeterServer) Unwrap() any {
	return m.store
}

var meterReadingPager = NewPageReader(func(r history.Record) (*meterpb.MeterReadingRecord, error) {
	v := &meterpb.MeterReading{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &meterpb.MeterReadingRecord{
		RecordTime:   timestamppb.New(r.CreateTime),
		MeterReading: v,
	}, nil
})

func (m *MeterServer) ListMeterReadingHistory(ctx context.Context, request *meterpb.ListMeterReadingHistoryRequest) (*meterpb.ListMeterReadingHistoryResponse, error) {
	page, size, nextToken, err := meterReadingPager.ListRecords(ctx, m.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &meterpb.ListMeterReadingHistoryResponse{
		TotalSize:           int32(size),
		NextPageToken:       nextToken,
		MeterReadingRecords: page,
	}, nil
}
