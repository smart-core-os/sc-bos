package alertpb

import (
	"context"

	"google.golang.org/grpc"
)

type ModelServer struct {
	UnimplementedAlertApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	RegisterAlertApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) ListAlerts(_ context.Context, request *ListAlertsRequest) (*ListAlertsResponse, error) {
	alert := m.model.GetAllAlerts()
	return &ListAlertsResponse{
		Alerts: alert,
	}, nil
}
