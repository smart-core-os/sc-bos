package serviceticketpb

import (
	"errors"

	"github.com/pborman/uuid"

	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type Model struct {
	tickets map[string]*Ticket // TicketId -> Ticket
	support *TicketSupport
}

func NewModel(opts ...resource.Option) *Model {
	m := &Model{}
	m.tickets = make(map[string]*Ticket)
	return m
}

func (m Model) addTicket(ticket *Ticket) *Ticket {
	id := uuid.NewUUID().String()
	ticket.Id = id
	m.tickets[id] = ticket
	return ticket
}

func (m Model) updateTicket(ticket *Ticket) (*Ticket, error) {
	if _, ok := m.tickets[ticket.Id]; !ok {
		return nil, errors.New("ticket not found")
	}
	m.tickets[ticket.Id] = ticket
	return ticket, nil
}

// SetSupport sets the serviceticketpb.TicketSupport to use in the ServiceTicketInfoServer.
func (m Model) SetSupport(s *TicketSupport) {
	m.support = s
}

func (m Model) GetTickets() (map[string]*Ticket, error) {
	return m.tickets, nil
}
