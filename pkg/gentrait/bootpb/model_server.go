package bootpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	bootproto "github.com/smart-core-os/sc-bos/pkg/proto/bootpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// ModelServer implements BootApiServer backed by a Model.
// The OnReboot callback is called when a Reboot RPC is received.
type ModelServer struct {
	bootproto.UnimplementedBootApiServer
	model    *Model
	OnReboot func(ctx context.Context, req *bootproto.RebootRequest) error
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) GetBootState(ctx context.Context, req *bootproto.GetBootStateRequest) (*bootproto.BootState, error) {
	return s.model.GetBootState(resource.WithReadMask(req.ReadMask)), nil
}

func (s *ModelServer) PullBootState(req *bootproto.PullBootStateRequest, server grpc.ServerStreamingServer[bootproto.PullBootStateResponse]) error {
	changes := s.model.PullBootState(server.Context(),
		resource.WithReadMask(req.ReadMask),
		resource.WithUpdatesOnly(req.UpdatesOnly),
	)
	for change := range changes {
		err := server.Send(&bootproto.PullBootStateResponse{
			Changes: []*bootproto.PullBootStateResponse_Change{
				{
					Name:       req.Name,
					ChangeTime: timestamppb.New(change.ChangeTime),
					BootState:  change.Value,
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

func (s *ModelServer) Reboot(ctx context.Context, req *bootproto.RebootRequest) (*bootproto.RebootResponse, error) {
	now := timestamppb.Now()
	if s.OnReboot != nil {
		if err := s.OnReboot(ctx, req); err != nil {
			return nil, err
		}
	}
	return &bootproto.RebootResponse{RebootTime: now}, nil
}
