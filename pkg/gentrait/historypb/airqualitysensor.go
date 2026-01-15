package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
)

type AirQualitySensorServer struct {
	airqualitysensorpb.UnimplementedAirQualitySensorHistoryServer
	store history.Store // payloads of *traits.AirQuality
}

func NewAirQualitySensorServer(store history.Store) *AirQualitySensorServer {
	return &AirQualitySensorServer{store: store}
}

func (m *AirQualitySensorServer) Register(server *grpc.Server) {
	airqualitysensorpb.RegisterAirQualitySensorHistoryServer(server, m)
}

func (m *AirQualitySensorServer) Unwrap() any {
	return m.store
}

var airQualityPager = NewPageReader(func(r history.Record) (*airqualitysensorpb.AirQualityRecord, error) {
	v := &traits.AirQuality{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &airqualitysensorpb.AirQualityRecord{
		RecordTime: timestamppb.New(r.CreateTime),
		AirQuality: v,
	}, nil
})

func (m *AirQualitySensorServer) ListAirQualityHistory(ctx context.Context, request *airqualitysensorpb.ListAirQualityHistoryRequest) (*airqualitysensorpb.ListAirQualityHistoryResponse, error) {
	page, size, nextToken, err := airQualityPager.ListRecords(ctx, m.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &airqualitysensorpb.ListAirQualityHistoryResponse{
		TotalSize:         int32(size),
		NextPageToken:     nextToken,
		AirQualityRecords: page,
	}, nil
}
