package lockunlockpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedLockUnlockApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterLockUnlockApiServer(server, s)
}

func (s *ModelServer) GetLockUnlock(_ context.Context, req *GetLockUnlockRequest) (*LockUnlock, error) {
	return s.model.GetLockUnlock(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) UpdateLockUnlock(_ context.Context, req *UpdateLockUnlockRequest) (*LockUnlock, error) {
	return s.model.UpdateLockUnlock(req.LockUnlock, resource.WithUpdateMask(req.UpdateMask))
}

func (s *ModelServer) PullLockUnlock(request *PullLockUnlockRequest, server LockUnlockApi_PullLockUnlockServer) error {
	for update := range s.model.PullLockUnlock(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullLockUnlockResponse{Changes: []*PullLockUnlockResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(update.ChangeTime),
			LockUnlock: update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
