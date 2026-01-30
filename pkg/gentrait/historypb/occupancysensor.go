package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
)

type OccupancySensorServer struct {
	occupancysensorpb.UnimplementedOccupancySensorHistoryServer
	store history.Store // payloads of *traits.Occupancy
}

func NewOccupancySensorServer(store history.Store) *OccupancySensorServer {
	return &OccupancySensorServer{store: store}
}

func (m *OccupancySensorServer) Register(server *grpc.Server) {
	occupancysensorpb.RegisterOccupancySensorHistoryServer(server, m)
}

func (m *OccupancySensorServer) Unwrap() any {
	return m.store
}

var occupancyPager = NewPageReader(func(r history.Record) (*occupancysensorpb.OccupancyRecord, error) {
	v := &traits.Occupancy{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &occupancysensorpb.OccupancyRecord{
		RecordTime: timestamppb.New(r.CreateTime),
		Occupancy:  v,
	}, nil
})

func (m *OccupancySensorServer) ListOccupancyHistory(ctx context.Context, request *occupancysensorpb.ListOccupancyHistoryRequest) (*occupancysensorpb.ListOccupancyHistoryResponse, error) {
	page, size, nextToken, err := occupancyPager.ListRecords(ctx, m.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &occupancysensorpb.ListOccupancyHistoryResponse{
		TotalSize:        int32(size),
		NextPageToken:    nextToken,
		OccupancyRecords: page,
	}, nil
}
