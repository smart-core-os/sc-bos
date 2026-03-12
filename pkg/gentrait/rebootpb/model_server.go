package rebootpb

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	rebootproto "github.com/smart-core-os/sc-bos/pkg/proto/rebootpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// ModelServer implements RebootApiServer backed by a Model.
// The OnReboot callback is called when a Reboot RPC is received.
type ModelServer struct {
	rebootproto.UnimplementedRebootApiServer
	model    *Model
	OnReboot func(ctx context.Context, req *rebootproto.RebootRequest) error
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) GetRebootState(ctx context.Context, req *rebootproto.GetRebootStateRequest) (*rebootproto.RebootState, error) {
	state := s.model.GetRebootState(resource.WithReadMask(req.ReadMask))
	// Compute uptime fresh on each call; don't store it in the model.
	if state != nil && state.BootTime != nil {
		uptime := time.Since(state.BootTime.AsTime())
		// Clone so we don't mutate the stored value.
		out := proto.Clone(state).(*rebootproto.RebootState)
		out.Uptime = durationpb.New(uptime)
		return out, nil
	}
	return state, nil
}

func (s *ModelServer) PullRebootState(req *rebootproto.PullRebootStateRequest, server grpc.ServerStreamingServer[rebootproto.PullRebootStateResponse]) error {
	changes := s.model.PullRebootState(server.Context(),
		resource.WithReadMask(req.ReadMask),
		resource.WithUpdatesOnly(req.UpdatesOnly),
	)
	for change := range changes {
		err := server.Send(&rebootproto.PullRebootStateResponse{
			Changes: []*rebootproto.PullRebootStateResponse_Change{
				{
					Name:        req.Name,
					ChangeTime:  timestamppb.New(change.ChangeTime),
					RebootState: change.Value,
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

func (s *ModelServer) Reboot(ctx context.Context, req *rebootproto.RebootRequest) (*rebootproto.RebootResponse, error) {
	now := timestamppb.Now()
	if s.OnReboot != nil {
		if err := s.OnReboot(ctx, req); err != nil {
			return nil, err
		}
	}
	return &rebootproto.RebootResponse{RebootTime: now}, nil
}
