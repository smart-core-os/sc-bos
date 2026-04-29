package energystoragepb

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialEnergyLevel(&EnergyLevel{}),
	WithEnergyLevelOption(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.1),
		cmp.TimeValueWithin(1*time.Second),
		cmp.DurationValueWithin(1*time.Second),
	))),
}

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithEnergyLevelOption configures the energyLevel resource of the model.
func WithEnergyLevelOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.energyLevelOpts = append(args.energyLevelOpts, opts...)
	})
}

// WithInitialEnergyLevel returns an option that configures the model to initialise with the given energy level.
func WithInitialEnergyLevel(energyLevel *EnergyLevel) resource.Option {
	return WithEnergyLevelOption(resource.WithInitialValue(energyLevel))
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	energyLevelOpts []resource.Option
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.energyLevelOpts = append(a.energyLevelOpts, opt)
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
