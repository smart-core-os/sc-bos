package bookingpb

import (
	"fmt"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions []resource.Option

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithBookingOption configures the booking resource of the model.
func WithBookingOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.bookingOpts = append(args.bookingOpts, opts...)
	})
}

// WithInitialBooking returns an option that configures the model to initialise with the given bookings.
// Can be used multiple times with bookings being additive.
// Creating a model with duplicate booking ids will panic.
// Calling this function with an empty booking id property will panic.
func WithInitialBooking(bookings ...*traits.Booking) resource.Option {
	opts := make([]resource.Option, len(bookings))
	for i, booking := range bookings {
		if booking.Id == "" {
			panic(fmt.Sprintf("booking at index %v has no Id property", i))
		}
		opts[i] = resource.WithInitialRecord(booking.Id, booking)
	}
	return WithBookingOption(opts...)
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	bookingOpts []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.bookingOpts = append(a.bookingOpts, opt)
	}
}

func modelOptionFunc(fn func(args *modelArgs)) ModelOption {
	return modelOption{resource.EmptyOption{}, fn}
}

type modelOption struct {
	resource.Option
	fn func(args *modelArgs)
}

func (m modelOption) applyModel(args *modelArgs) {
	m.fn(args)
}
