package accesspb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	traits.UnimplementedAccessApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	traits.RegisterAccessApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetLastAccessAttempt(ctx context.Context, request *traits.GetLastAccessAttemptRequest) (*traits.AccessAttempt, error) {
	return m.model.GetLastAccessAttempt(resource.WithReadMask(request.GetReadMask()))
}

func (m *ModelServer) PullAccessAttempts(request *traits.PullAccessAttemptsRequest, server traits.AccessApi_PullAccessAttemptsServer) error {
	for change := range m.model.PullAccessAttempts(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly())) {
		msg := &traits.PullAccessAttemptsResponse{Changes: []*traits.PullAccessAttemptsResponse_Change{{
			Name:          request.Name,
			ChangeTime:    timestamppb.New(change.ChangeTime),
			AccessAttempt: change.Value,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
