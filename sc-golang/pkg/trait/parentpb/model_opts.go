package parentpb

import (
	"fmt"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/time/clock"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	resource.WithClock(clock.Real()),
}

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithChildrenOption configures the children resource of the model.
func WithChildrenOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.childrenOpts = append(args.childrenOpts, opts...)
	})
}

// WithInitialChildren returns an option that configures the model to initialise with the given children.
// Can be called multiple times to add more children.
// Panics if any child has no name.
// Panics if any child's traits are not sorted.
func WithInitialChildren(children ...*traits.Child) resource.Option {
	opts := make([]resource.Option, len(children))
	for i, child := range children {
		if child.Name == "" {
			panic(fmt.Sprintf("child at index %d has no name", i))
		}
		opts[i] = resource.WithInitialRecord(child.Name, child)
	}
	return WithChildrenOption(opts...)
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	childrenOpts []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.childrenOpts = append(a.childrenOpts, opt)
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
