package lightpb

import (
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialBrightness(&traits.Brightness{}),
}

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithBrightnessOption configures the brightness resource of the model.
func WithBrightnessOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.brightnessOpts = append(args.brightnessOpts, opts...)
	})
}

// WithInitialBrightness returns an option that configures the model to initialise with the given brightness.
func WithInitialBrightness(brightness *traits.Brightness) resource.Option {
	return WithBrightnessOption(resource.WithInitialValue(brightness))
}

// WithPreset instructs the model to set the light to the given level when preset p is selected.
func WithPreset(levelPercent float32, p *traits.LightPreset) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.presets = append(args.presets, preset{p, levelPercent})
	})
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	brightnessOpts []resource.Option
	presets        []preset
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.brightnessOpts = append(a.brightnessOpts, opt)
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
