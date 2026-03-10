package electricpb

import (
	"fmt"
	"math/rand"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/time/clock"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialDemand(&traits.ElectricDemand{
		Current: 0,
		Voltage: &defaultInitialVoltage,
		Rating:  13,
	}),
	WithInitialActiveMode(&traits.ElectricMode{}),
	WithDemandOption(resource.WithMessageEquivalence(cmp.Equal(cmp.FloatValueApprox(0, 0.01)))),
	WithActiveModeOption(resource.WithNoDuplicates()),
	WithModeOption(resource.WithNoDuplicates()),
	WithClock(clock.Real()),
	WithRNG(rand.New(rand.NewSource(rand.Int63()))),
}
var defaultInitialVoltage float32 = 240

// ModelOption defined the base type for all options that apply to this traits model.
type ModelOption interface {
	resource.Option
	applyModel(args *modelArgs)
}

// WithDemandOption configures the demand resource of the model.
func WithDemandOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.demandOpts = append(args.demandOpts, opts...)
	})
}

// WithActiveModeOption configures the activeMode resource of the model.
func WithActiveModeOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.activeModeOpts = append(args.activeModeOpts, opts...)
	})
}

// WithModeOption configures the mode resource of the model.
func WithModeOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.modeOpts = append(args.modeOpts, opts...)
	})
}

// WithInitialDemand returns an option that configures the model to initialise with the given demand.
func WithInitialDemand(demand *traits.ElectricDemand) resource.Option {
	return WithDemandOption(resource.WithInitialValue(demand))
}

// WithInitialActiveMode returns an option that configures the model to initialise with the given active mode.
func WithInitialActiveMode(activeMode *traits.ElectricMode) resource.Option {
	return WithActiveModeOption(resource.WithInitialValue(activeMode))
}

// WithInitialMode returns an option that configures the model to initialise with the given modes.
// Can be used multiple times with modes being additive.
// Creating a model with duplicate mode ids will panic.
// Calling this function with an empty mode id property will panic.
func WithInitialMode(mode ...*traits.ElectricMode) resource.Option {
	opts := make([]resource.Option, len(mode))
	for i, m := range mode {
		if m.Id == "" {
			panic(fmt.Sprintf("mode at index %v has no Id property", i))
		}
		opts[i] = resource.WithInitialRecord(m.Id, m)
	}
	return WithModeOption(opts...)
}

// WithClock returns an option that configures the model to use the given clock for all resources.
// Overrides resource.WithClock() if used before this option.
// Combining this with resource.WithClock() is not recommended.
func WithClock(clock clock.Clock) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.demandOpts = append(args.demandOpts, resource.WithClock(clock))
		args.activeModeOpts = append(args.activeModeOpts, resource.WithClock(clock))
		args.modeOpts = append(args.modeOpts, resource.WithClock(clock))
		args.clock = clock
	})
}

// WithRNG returns an option that configures the model to use the given random number generator.
// Overrides resource.WithRNG() if used before this option.
func WithRNG(rng *rand.Rand) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.demandOpts = append(args.demandOpts, resource.WithRNG(rng))
		args.activeModeOpts = append(args.activeModeOpts, resource.WithRNG(rng))
		args.modeOpts = append(args.modeOpts, resource.WithRNG(rng))
		args.rng = rng
	})
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	demandOpts     []resource.Option
	activeModeOpts []resource.Option
	modeOpts       []resource.Option
	clock          clock.Clock
	rng            *rand.Rand
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(ModelOption); ok {
			v.applyModel(a)
			continue
		}
		a.demandOpts = append(a.demandOpts, opt)
		a.activeModeOpts = append(a.activeModeOpts, opt)
		a.modeOpts = append(a.modeOpts, opt)
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
