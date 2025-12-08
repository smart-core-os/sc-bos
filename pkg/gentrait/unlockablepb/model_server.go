package unlockablepb

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/gen"
)

type ModelServer struct {
	gen.UnimplementedUnlockableAPIServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	gen.RegisterUnlockableAPIServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetLockers(context.Context, *gen.ListUnlockablesRequest) (*gen.ListUnlockablesResponse, error) {
	return &gen.ListUnlockablesResponse{UnlockableBanks: m.model.unlockableBanks}, nil
}
