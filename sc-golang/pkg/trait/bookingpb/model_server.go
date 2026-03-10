package bookingpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types/time"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	timepb "github.com/smart-core-os/sc-golang/pkg/time"
)

type ModelServer struct {
	traits.UnimplementedBookingApiServer

	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	traits.RegisterBookingApiServer(server, m)
}

func (m *ModelServer) ListBookings(_ context.Context, request *traits.ListBookingsRequest) (*traits.ListBookingsResponse, error) {
	opts := []resource.ReadOption{
		resource.WithReadMask(request.ReadMask),
	}
	if request.BookingIntersects != nil {
		opts = append(opts, resource.WithInclude(func(_ string, item proto.Message) bool {
			if item == nil {
				return false
			}
			itemVal := item.(*traits.Booking)
			return timepb.PeriodsIntersect(itemVal.Booked, request.BookingIntersects)
		}))
	}
	bookings := m.model.ListBookings(opts...)
	return &traits.ListBookingsResponse{Bookings: bookings}, nil
}

func (m *ModelServer) CheckInBooking(_ context.Context, request *traits.CheckInBookingRequest) (*traits.CheckInBookingResponse, error) {
	t := request.Time
	if t == nil {
		t = serverTimestamp() // todo: use the resource clock
	}
	mask, err := fieldmaskpb.New(&traits.Booking{}, "check_in.start_time")
	if err != nil {
		return nil, err // panic?
	}
	checkInBooking := &traits.Booking{
		Id: request.BookingId,
		CheckIn: &time.Period{
			StartTime: t,
		},
	}
	_, err = m.model.UpdateBooking(checkInBooking, resource.WithUpdateMask(mask))
	if err != nil {
		return nil, err
	}
	return &traits.CheckInBookingResponse{}, nil
}

func (m *ModelServer) CheckOutBooking(_ context.Context, request *traits.CheckOutBookingRequest) (*traits.CheckOutBookingResponse, error) {
	t := request.Time
	if t == nil {
		t = serverTimestamp() // todo: use the resource clock
	}
	mask, err := fieldmaskpb.New(&traits.Booking{}, "check_in.end_time")
	if err != nil {
		return nil, err // panic?
	}
	checkInBooking := &traits.Booking{
		Id: request.BookingId,
		CheckIn: &time.Period{
			EndTime: t,
		},
	}
	_, err = m.model.UpdateBooking(checkInBooking, resource.WithUpdateMask(mask))
	if err != nil {
		return nil, err
	}
	return &traits.CheckOutBookingResponse{}, nil
}

func (m *ModelServer) CreateBooking(_ context.Context, request *traits.CreateBookingRequest) (*traits.CreateBookingResponse, error) {
	b := request.GetBooking()
	if b == nil {
		b = &traits.Booking{}
	}

	booking, err := m.model.CreateBooking(b)
	if err != nil {
		return nil, err
	}
	return &traits.CreateBookingResponse{BookingId: booking.Id}, nil
}

func (m *ModelServer) UpdateBooking(ctx context.Context, request *traits.UpdateBookingRequest) (*traits.UpdateBookingResponse, error) {
	booking, err := m.model.UpdateBooking(request.Booking, resource.WithUpdateMask(request.UpdateMask))
	if err != nil {
		return nil, err
	}
	return &traits.UpdateBookingResponse{Booking: booking}, nil
}

func (m *ModelServer) PullBookings(request *traits.ListBookingsRequest, server traits.BookingApi_PullBookingsServer) error {
	opts := []resource.ReadOption{
		resource.WithReadMask(request.ReadMask),
		resource.WithUpdatesOnly(request.UpdatesOnly),
	}
	if request.BookingIntersects != nil {
		opts = append(opts, resource.WithInclude(func(id string, item proto.Message) bool {
			if item == nil {
				return false
			}
			itemVal := item.(*traits.Booking)
			return timepb.PeriodsIntersect(itemVal.Booked, request.BookingIntersects)
		}))
	}

	for change := range m.model.PullBookings(server.Context(), opts...) {
		err := server.Send(&traits.PullBookingsResponse{Changes: []*traits.PullBookingsResponse_Change{
			{
				Name:       request.Name,
				ChangeTime: timestamppb.New(change.ChangeTime),
				Type:       change.ChangeType,
				OldValue:   change.OldValue,
				NewValue:   change.NewValue,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
