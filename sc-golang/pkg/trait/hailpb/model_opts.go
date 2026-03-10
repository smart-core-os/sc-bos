package hailpb

import (
	"time"

	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithKeepAlive(30 * time.Second),
}

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithKeepAlive configures the minimum time a Hail will live after it has reached the ARRIVED state before it is
// deleted. It may last longer than this time.
//
// Negative durations imply hails are never automatically removed.
func WithKeepAlive(keepAlive time.Duration) ModelOption {
	return modelOptionFunc(func(args *modelArgs) {
		args.keepAlive = keepAlive
	})
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	keepAlive time.Duration

	hailsOptions []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.hailsOptions = append(a.hailsOptions, opt)
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
