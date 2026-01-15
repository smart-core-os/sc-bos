package serviceticketpb

import (
	"errors"

	"github.com/pborman/uuid"

	"github.com/smart-core-os/sc-bos/pkg/proto/serviceticketpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	tickets map[string]*serviceticketpb.Ticket // TicketId -> Ticket
	support *serviceticketpb.TicketSupport
}

func NewModel(opts ...resource.Option) *Model {
	m := &Model{}
	m.tickets = make(map[string]*serviceticketpb.Ticket)
	return m
}

func (m Model) addTicket(ticket *serviceticketpb.Ticket) *serviceticketpb.Ticket {
	id := uuid.NewUUID().String()
	ticket.Id = id
	m.tickets[id] = ticket
	return ticket
}

func (m Model) updateTicket(ticket *serviceticketpb.Ticket) (*serviceticketpb.Ticket, error) {
	if _, ok := m.tickets[ticket.Id]; !ok {
		return nil, errors.New("ticket not found")
	}
	m.tickets[ticket.Id] = ticket
	return ticket, nil
}

// SetSupport sets the serviceticketpb.TicketSupport to use in the ServiceTicketInfoServer.
func (m Model) SetSupport(s *serviceticketpb.TicketSupport) {
	m.support = s
}

func (m Model) GetTickets() (map[string]*serviceticketpb.Ticket, error) {
	return m.tickets, nil
}
