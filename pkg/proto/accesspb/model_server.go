package accesspb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type ModelServer struct {
	UnimplementedAccessApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	RegisterAccessApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetLastAccessAttempt(ctx context.Context, request *GetLastAccessAttemptRequest) (*AccessAttempt, error) {
	return m.model.GetLastAccessAttempt(resource.WithReadMask(request.GetReadMask()))
}

func (m *ModelServer) PullAccessAttempts(request *PullAccessAttemptsRequest, server AccessApi_PullAccessAttemptsServer) error {
	for change := range m.model.PullAccessAttempts(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly())) {
		msg := &PullAccessAttemptsResponse{Changes: []*PullAccessAttemptsResponse_Change{{
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
