package openclosepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedOpenCloseApiServer
	UnimplementedOpenCloseInfoServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterOpenCloseApiServer(server, s)
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) GetPositions(_ context.Context, request *GetOpenClosePositionsRequest) (*OpenClosePositions, error) {
	return s.model.GetPositions(resource.WithReadMask(request.GetReadMask()))
}

func (s *ModelServer) UpdatePositions(_ context.Context, request *UpdateOpenClosePositionsRequest) (*OpenClosePositions, error) {
	return s.model.UpdatePositions(request.GetStates(), resource.WithUpdateMask(request.GetUpdateMask()))
}

func (s *ModelServer) PullPositions(request *PullOpenClosePositionsRequest, server OpenCloseApi_PullPositionsServer) error {
	for change := range s.model.PullPositions(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly())) {
		msg := &PullOpenClosePositionsResponse{Changes: []*PullOpenClosePositionsResponse_Change{{
			Name:              request.Name,
			ChangeTime:        timestamppb.New(change.ChangeTime),
			OpenClosePosition: change.Positions,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *ModelServer) DescribePositions(_ context.Context, _ *DescribePositionsRequest) (*PositionsSupport, error) {
	support := &PositionsSupport{
		ResourceSupport: &typespb.ResourceSupport{
			Readable: true, Writable: true, Observable: true,
			PullSupport: typespb.PullSupport_PULL_SUPPORT_NATIVE,
		},
		SupportsStop:          true,
		Presets:               s.model.ListPresets(),
		OpenPercentAttributes: s.model.OpenPercentAttributes(),
	}
	return support, nil
}
