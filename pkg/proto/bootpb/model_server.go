package bootpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// ModelServer implements BootApiServer backed by a Model.
// The OnReboot callback is called when a Reboot RPC is received.
type ModelServer struct {
	UnimplementedBootApiServer
	model    *Model
	OnReboot func(ctx context.Context, req *RebootRequest) error
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Register(server *grpc.Server) {
	RegisterBootApiServer(server, s)
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) GetBootState(ctx context.Context, req *GetBootStateRequest) (*BootState, error) {
	return s.model.GetBootState(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) PullBootState(req *PullBootStateRequest, server grpc.ServerStreamingServer[PullBootStateResponse]) error {
	changes := s.model.PullBootState(server.Context(),
		resource.WithReadMask(req.ReadMask),
		resource.WithUpdatesOnly(req.UpdatesOnly),
	)
	for change := range changes {
		err := server.Send(&PullBootStateResponse{
			Changes: []*PullBootStateResponse_Change{
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

func (s *ModelServer) Reboot(ctx context.Context, req *RebootRequest) (*RebootResponse, error) {
	if s.OnReboot != nil {
		if err := s.OnReboot(ctx, req); err != nil {
			return nil, err
		}
	}
	// Capture the timestamp after the reboot is accepted, before the response is sent.
	// The actual reboot will occur after the caller receives this response.
	return &RebootResponse{RebootTime: timestamppb.Now()}, nil
}
