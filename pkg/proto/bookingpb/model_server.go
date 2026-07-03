package bookingpb

import (
	"context"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	timepb2 "github.com/smart-core-os/sc-bos/pkg/proto/timepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	timepb "github.com/smart-core-os/sc-bos/pkg/util/time"
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
	pageToken := &typespb.PageToken{}
	if err := decodePageToken(request.GetPageToken(), pageToken); err != nil {
		return nil, err
	}

	lastKey := pageToken.GetLastResourceName() // the id of the last booking we sent
	pageSize := capPageSize(int(request.GetPageSize()))

	// Fetch the full (unmasked) set so pagination keys off booking id regardless of the read mask.
	// The read mask is applied to the returned page below.
	var opts []resource.ReadOption
	if request.BookingIntersects != nil {
		opts = append(opts, resource.WithInclude(func(_ string, item proto.Message) bool {
			if item == nil {
				return false
			}
			itemVal := item.(*Booking)
			return timepb.PeriodsIntersect(itemVal.Booked, request.BookingIntersects)
		}))
	}
	all := m.model.ListBookings(opts...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Id < all[j].Id
	})

	nextIndex := 0
	if lastKey != "" {
		nextIndex = sort.Search(len(all), func(i int) bool {
			return all[i].Id >= lastKey
		})
		if nextIndex < len(all) && all[nextIndex].Id == lastKey {
			nextIndex++
		}
	}

	result := &ListBookingsResponse{
		TotalSize: int32(len(all)),
	}
	upperBound := nextIndex + pageSize
	if upperBound > len(all) {
		upperBound = len(all)
		pageToken = nil
	} else {
		pageToken.PageStart = &typespb.PageToken_LastResourceName{
			LastResourceName: all[upperBound-1].Id,
		}
	}

	var err error
	result.NextPageToken, err = encodePageToken(pageToken)
	if err != nil {
		return nil, err
	}

	page := all[nextIndex:upperBound]
	mask := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))
	result.Bookings = make([]*Booking, len(page))
	for i, b := range page {
		result.Bookings[i] = mask.FilterClone(b).(*Booking)
	}

	return result, nil
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
		CheckIn: &timepb2.Period{
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
		CheckIn: &timepb2.Period{
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
