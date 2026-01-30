package serviceticketpb

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/serviceticketpb"
)

type ModelServer struct {
	serviceticketpb.UnimplementedServiceTicketApiServer
	serviceticketpb.UnimplementedServiceTicketInfoServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	serviceticketpb.RegisterServiceTicketApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) CreateTicket(_ context.Context, req *serviceticketpb.CreateTicketRequest) (*serviceticketpb.Ticket, error) {
	return m.model.addTicket(req.Ticket), nil
}

func (m *ModelServer) UpdateTicket(_ context.Context, req *serviceticketpb.UpdateTicketRequest) (*serviceticketpb.Ticket, error) {
	return m.model.updateTicket(req.Ticket)
}

func (m *ModelServer) DescribeTicket(context.Context, *serviceticketpb.DescribeTicketRequest) (*serviceticketpb.TicketSupport, error) {
	return m.model.support, nil
}
