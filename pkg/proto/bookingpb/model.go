package bookingpb

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// Model models the Booking trait.
type Model struct {
	bookings *resource.Collection // Of *traits.Booking
}

// NewModel creates a new Model without any bookings.
func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		bookings: resource.NewCollection(args.bookingOpts...),
	}
}

func (m *Model) ListBookings(opts ...resource.ReadOption) []*Booking {
	msgs := m.bookings.List(opts...)
	res := make([]*Booking, len(msgs))
	for i, msg := range msgs {
		res[i] = msg.(*Booking)
	}
	return res
}

func (m *Model) CreateBooking(booking *Booking) (*Booking, error) {
	msg, err := m.bookings.Add(booking.Id, booking, resource.WithGenIDIfAbsent(), resource.WithIDCallback(func(id string) {
		booking.Id = id
	}))
	if msg == nil {
		return nil, err
	}
	return msg.(*Booking), err
}

func (m *Model) UpdateBooking(booking *Booking, opts ...resource.WriteOption) (*Booking, error) {
	if booking.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing booking.id")
	}

	newVal, err := m.bookings.Update(booking.Id, booking, opts...)
	if newVal == nil {
		return nil, err
	}
	return newVal.(*Booking), err
}

//goland:noinspection GoNameStartsWithPackageName
type BookingChange struct {
	ChangeTime time.Time
	ChangeType typespb.ChangeType

	OldValue, NewValue *Booking
}

func (m *Model) PullBookings(ctx context.Context, opts ...resource.ReadOption) <-chan BookingChange {
	send := make(chan BookingChange)

	go func() {
		defer close(send)
		for change := range m.bookings.Pull(ctx, opts...) {
			event := BookingChange{
				ChangeTime: change.ChangeTime,
				ChangeType: change.ChangeType,
			}
			if change.OldValue != nil {
				event.OldValue = change.OldValue.(*Booking)
			}
			if change.NewValue != nil {
				event.NewValue = change.NewValue.(*Booking)
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
