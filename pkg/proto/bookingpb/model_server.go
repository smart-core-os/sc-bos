package bookingpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	timepb "github.com/smart-core-os/sc-bos/pkg/util/time"
	"github.com/smart-core-os/sc-bos/sc-api/go/types/time"
)

type ModelServer struct {
	UnimplementedBookingApiServer

	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterBookingApiServer(server, m)
}

func (m *ModelServer) ListBookings(_ context.Context, request *ListBookingsRequest) (*ListBookingsResponse, error) {
	opts := []resource.ReadOption{
		resource.WithReadMask(request.ReadMask),
	}
	if request.BookingIntersects != nil {
		opts = append(opts, resource.WithInclude(func(_ string, item proto.Message) bool {
			if item == nil {
				return false
			}
			itemVal := item.(*Booking)
			return timepb.PeriodsIntersect(itemVal.Booked, request.BookingIntersects)
		}))
	}
	bookings := m.model.ListBookings(opts...)
	return &ListBookingsResponse{Bookings: bookings}, nil
}

func (m *ModelServer) CheckInBooking(_ context.Context, request *CheckInBookingRequest) (*CheckInBookingResponse, error) {
	t := request.Time
	if t == nil {
		t = serverTimestamp() // todo: use the resource clock
	}
	mask, err := fieldmaskpb.New(&Booking{}, "check_in.start_time")
	if err != nil {
		return nil, err // panic?
	}
	checkInBooking := &Booking{
		Id: request.BookingId,
		CheckIn: &time.Period{
			StartTime: t,
		},
	}
	_, err = m.model.UpdateBooking(checkInBooking, resource.WithUpdateMask(mask))
	if err != nil {
		return nil, err
	}
	return &CheckInBookingResponse{}, nil
}

func (m *ModelServer) CheckOutBooking(_ context.Context, request *CheckOutBookingRequest) (*CheckOutBookingResponse, error) {
	t := request.Time
	if t == nil {
		t = serverTimestamp() // todo: use the resource clock
	}
	mask, err := fieldmaskpb.New(&Booking{}, "check_in.end_time")
	if err != nil {
		return nil, err // panic?
	}
	checkInBooking := &Booking{
		Id: request.BookingId,
		CheckIn: &time.Period{
			EndTime: t,
		},
	}
	_, err = m.model.UpdateBooking(checkInBooking, resource.WithUpdateMask(mask))
	if err != nil {
		return nil, err
	}
	return &CheckOutBookingResponse{}, nil
}

func (m *ModelServer) CreateBooking(_ context.Context, request *CreateBookingRequest) (*CreateBookingResponse, error) {
	b := request.GetBooking()
	if b == nil {
		b = &Booking{}
	}

	booking, err := m.model.CreateBooking(b)
	if err != nil {
		return nil, err
	}
	return &CreateBookingResponse{BookingId: booking.Id}, nil
}

func (m *ModelServer) UpdateBooking(ctx context.Context, request *UpdateBookingRequest) (*UpdateBookingResponse, error) {
	booking, err := m.model.UpdateBooking(request.Booking, resource.WithUpdateMask(request.UpdateMask))
	if err != nil {
		return nil, err
	}
	return &UpdateBookingResponse{Booking: booking}, nil
}

func (m *ModelServer) PullBookings(request *ListBookingsRequest, server BookingApi_PullBookingsServer) error {
	opts := []resource.ReadOption{
		resource.WithReadMask(request.ReadMask),
		resource.WithUpdatesOnly(request.UpdatesOnly),
	}
	if request.BookingIntersects != nil {
		opts = append(opts, resource.WithInclude(func(id string, item proto.Message) bool {
			if item == nil {
				return false
			}
			itemVal := item.(*Booking)
			return timepb.PeriodsIntersect(itemVal.Booked, request.BookingIntersects)
		}))
	}

	for change := range m.model.PullBookings(server.Context(), opts...) {
		err := server.Send(&PullBookingsResponse{Changes: []*PullBookingsResponse_Change{
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
