package hailpb

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model describes the data structure needed to implement the Hail trait.
type Model struct {
	hails *resource.Collection // of *traits.Hail

	keepAlive time.Duration

	gcTicket chan struct{} // size-1 chan that has an item in when a gc run should succeed
}

// NewModel creates a new model.
func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	gcTicket := make(chan struct{}, 1)
	gcTicket <- struct{}{} // prime the ticket machine so the gc runs
	return &Model{
		hails:     resource.NewCollection(args.hailsOptions...),
		keepAlive: args.keepAlive,

		gcTicket: gcTicket,
	}
}

func (m *Model) CreateHail(hail *traits.Hail) (*traits.Hail, error) {
	defer m.gc()
	return castReturn(m.hails.Add("", hail, resource.WithGenIDIfAbsent(), resource.WithIDCallback(func(id string) {
		hail.Id = id
	})))
}

func (m *Model) GetHail(id string, opts ...resource.ReadOption) (*traits.Hail, bool) {
	msg, exists := m.hails.Get(id, opts...)
	if msg == nil {
		return nil, exists
	}
	return msg.(*traits.Hail), exists
}

func (m *Model) UpdateHail(hail *traits.Hail, opts ...resource.WriteOption) (*traits.Hail, error) {
	if hail.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing ID")
	}
	msg, err := m.hails.Update(hail.Id, hail, opts...)
	return castReturn(msg, err)
}

func (m *Model) DeleteHail(id string, opts ...resource.WriteOption) (*traits.Hail, error) {
	return castReturn(m.hails.Delete(id, opts...))
}

//goland:noinspection GoNameStartsWithPackageName
type HailChange struct {
	ChangeTime time.Time
	Value      *traits.Hail
}

// PullHail subscribes to changes in a single hail.
// The returned channel is closed when ctx is Done or the hail identified by id is deleted.
func (m *Model) PullHail(ctx context.Context, id string, opts ...resource.ReadOption) <-chan HailChange {
	send := make(chan HailChange)
	go func() {
		defer close(send)
		for change := range m.hails.PullID(ctx, id, opts...) {
			select {
			case <-ctx.Done():
				return
			case send <- HailChange{ChangeTime: change.ChangeTime, Value: change.Value.(*traits.Hail)}:
			}
		}
	}()
	return send
}

func (m *Model) ListHails(opts ...resource.ReadOption) []*traits.Hail {
	msgs := m.hails.List(opts...)
	hails := make([]*traits.Hail, len(msgs))
	for i, msg := range msgs {
		hails[i] = msg.(*traits.Hail)
	}
	return hails
}

type HailsChange struct {
	ChangeType types.ChangeType
	ChangeTime time.Time
	OldValue   *traits.Hail
	NewValue   *traits.Hail
}

func (m *Model) PullHails(ctx context.Context, opts ...resource.ReadOption) <-chan HailsChange {
	send := make(chan HailsChange)
	go func() {
		defer close(send)
		for change := range m.hails.Pull(ctx, opts...) {
			oldVal, newVal := castChange(change)
			event := HailsChange{
				ChangeType: change.ChangeType,
				ChangeTime: change.ChangeTime,
				OldValue:   oldVal,
				NewValue:   newVal,
			}
			select {
			case <-ctx.Done():
				return
			case send <- event:
			}
		}
	}()
	return send
}

func (m *Model) gc() {
	if m.keepAlive < 0 {
		return
	}

	select {
	case <-m.gcTicket:
		// there is a ticket
	default:
		return // no ticket
	}

	// setup the next gc
	time.AfterFunc(m.keepAlive, func() {
		m.gcTicket <- struct{}{}
	})

	now := m.hails.Clock().Now()
	t := now.Add(-m.keepAlive)
	for _, msg := range m.hails.List() {
		hail := msg.(*traits.Hail)
		if arrivedBefore(hail, t) {
			_, _ = m.hails.Delete(hail.Id, resource.WithAllowMissing(true), resource.WithExpectedValue(hail))
		}
	}
}

func arrivedBefore(hail *traits.Hail, t time.Time) bool {
	if hail.ArriveTime == nil {
		return false
	}
	arriveTime := hail.ArriveTime.AsTime()
	return arriveTime.Before(t)
}

func castReturn(msg proto.Message, err error) (*traits.Hail, error) {
	if msg == nil {
		return nil, err
	}
	return msg.(*traits.Hail), err
}

func castChange(change *resource.CollectionChange) (old, new *traits.Hail) {
	if change.OldValue != nil {
		old = change.OldValue.(*traits.Hail)
	}
	if change.NewValue != nil {
		new = change.NewValue.(*traits.Hail)
	}
	return
}
