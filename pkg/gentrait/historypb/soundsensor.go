package historypb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
)

type SoundSensorServer struct {
	soundsensorpb.UnimplementedSoundSensorHistoryServer
	store history.Store // payloads of *traits.AirQuality
}

func NewSoundSensorServer(store history.Store) *SoundSensorServer {
	return &SoundSensorServer{store: store}
}

func (m *SoundSensorServer) Register(server *grpc.Server) {
	soundsensorpb.RegisterSoundSensorHistoryServer(server, m)
}

func (m *SoundSensorServer) Unwrap() any {
	return m.store
}

var soundSensorPager = NewPageReader(func(r history.Record) (*soundsensorpb.SoundLevelRecord, error) {
	v := &soundsensorpb.SoundLevel{}
	err := proto.Unmarshal(r.Payload, v)
	if err != nil {
		return nil, err
	}
	return &soundsensorpb.SoundLevelRecord{
		RecordTime: timestamppb.New(r.CreateTime),
		SoundLevel: v,
	}, nil
})

func (m *SoundSensorServer) ListSoundLevelHistory(ctx context.Context, request *soundsensorpb.ListSoundLevelHistoryRequest) (*soundsensorpb.ListSoundLevelHistoryResponse, error) {
	page, size, nextToken, err := soundSensorPager.ListRecords(ctx, m.store, request.Period, int(request.PageSize), request.PageToken, request.OrderBy)
	if err != nil {
		return nil, err
	}

	return &soundsensorpb.ListSoundLevelHistoryResponse{
		TotalSize:         int32(size),
		NextPageToken:     nextToken,
		SoundLevelRecords: page,
	}, nil
}
