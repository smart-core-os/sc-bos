package alert

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
)

type ModelServer struct {
	alertpb.UnimplementedAlertApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	alertpb.RegisterAlertApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) ListAlerts(_ context.Context, request *alertpb.ListAlertsRequest) (*alertpb.ListAlertsResponse, error) {
	alert := m.model.GetAllAlerts()
	return &alertpb.ListAlertsResponse{
		Alerts: alert,
	}, nil
}
