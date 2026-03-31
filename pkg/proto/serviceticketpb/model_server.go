package serviceticketpb

import (
	"context"

	"google.golang.org/grpc"
)

type ModelServer struct {
	UnimplementedServiceTicketApiServer
	UnimplementedServiceTicketInfoServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	RegisterServiceTicketApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) CreateTicket(_ context.Context, req *CreateTicketRequest) (*Ticket, error) {
	return m.model.addTicket(req.Ticket), nil
}

func (m *ModelServer) UpdateTicket(_ context.Context, req *UpdateTicketRequest) (*Ticket, error) {
	return m.model.updateTicket(req.Ticket)
}

func (m *ModelServer) DescribeTicket(context.Context, *DescribeTicketRequest) (*TicketSupport, error) {
	return m.model.support, nil
}
