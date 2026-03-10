package fanspeedpb

import (
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialFanSpeed(&traits.FanSpeed{
		Percentage: DefaultPresets[0].Percentage,
		Preset:     DefaultPresets[0].Name,
		Direction:  traits.FanSpeed_FORWARD,
	}),
	WithFanSpeedOption(resource.WithMessageEquivalence(cmp.Equal(cmp.FloatValueApprox(0, 0.01)))),
	WithPresets(DefaultPresets...),
}

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithFanSpeedOption configures the fanSpeed resource of the model.
func WithFanSpeedOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.fanSpeedOpts = append(args.fanSpeedOpts, opts...)
	})
}

// WithInitialFanSpeed returns an option that configures the model to initialise with the given fan speed.
func WithInitialFanSpeed(fanSpeed *traits.FanSpeed) resource.Option {
	return WithFanSpeedOption(resource.WithInitialValue(fanSpeed))
}

// WithPresets returns an option that configures the model to use the given presets.
func WithPresets(presets ...Preset) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.presets = presets
	})
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	fanSpeedOpts []resource.Option
	presets      []Preset
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.fanSpeedOpts = append(a.fanSpeedOpts, opt)
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
