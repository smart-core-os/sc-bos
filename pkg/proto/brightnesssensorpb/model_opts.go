package brightnesssensorpb

import (
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialAmbientBrightness(&traits.AmbientBrightness{}),
	WithAmbientBrightnessOption(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.1),
	))),
}

// ModelOption defines the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithAmbientBrightnessOption configures the ambientBrightness resource of the model.
func WithAmbientBrightnessOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.ambientBrightnessOpts = append(args.ambientBrightnessOpts, opts...)
	})
}

// WithInitialAmbientBrightness returns an option that configures the model to initialise with the given ambient brightness.
func WithInitialAmbientBrightness(ambientBrightness *traits.AmbientBrightness) resource.Option {
	return WithAmbientBrightnessOption(resource.WithInitialValue(ambientBrightness))
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	ambientBrightnessOpts []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.ambientBrightnessOpts = append(a.ambientBrightnessOpts, opt)
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
