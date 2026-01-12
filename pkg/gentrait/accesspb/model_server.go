package accesspb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type ModelServer struct {
	accesspb.UnimplementedAccessApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	accesspb.RegisterAccessApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetLastAccessAttempt(ctx context.Context, request *accesspb.GetLastAccessAttemptRequest) (*accesspb.AccessAttempt, error) {
	return m.model.GetLastAccessAttempt(resource.WithReadMask(request.GetReadMask()))
}

func (m *ModelServer) PullAccessAttempts(request *accesspb.PullAccessAttemptsRequest, server accesspb.AccessApi_PullAccessAttemptsServer) error {
	for change := range m.model.PullAccessAttempts(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly())) {
		msg := &accesspb.PullAccessAttemptsResponse{Changes: []*accesspb.PullAccessAttemptsResponse_Change{{
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
