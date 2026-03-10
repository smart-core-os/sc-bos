package enterleavesensorpb

import (
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialEnterLeaveEvent(&traits.EnterLeaveEvent{
		EnterTotal: &zero,
		LeaveTotal: &zero,
	}),
}

var zero int32

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithEnterLeaveEventOption configures the enterLeaveEvent resource of the model.
func WithEnterLeaveEventOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.enterLeaveEventsOpts = append(args.enterLeaveEventsOpts, opts...)
	})
}

// WithInitialEnterLeaveEvent returns an option that configures the model to initialise with the given enter leave event.
func WithInitialEnterLeaveEvent(enterLeaveEvent *traits.EnterLeaveEvent) resource.Option {
	return WithEnterLeaveEventOption(resource.WithInitialValue(enterLeaveEvent))
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	enterLeaveEventsOpts []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.enterLeaveEventsOpts = append(a.enterLeaveEventsOpts, opt)
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
