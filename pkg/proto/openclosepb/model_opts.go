package openclosepb

import (
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// DefaultModelOptions holds the default options for the model.
var DefaultModelOptions = []resource.Option{
	WithInitialPositions(), // no positions
}

// WithPositionsOption configures options for the position resource.
func WithPositionsOption(opts ...resource.Option) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.positionsOpts = append(args.positionsOpts, opts...)
	})
}

// WithInitialPositions returns an option that configures the positions resource to initialise with the given positions.
// This option should only be used once per model.
func WithInitialPositions(positions ...*OpenClosePosition) resource.Option {
	var opts []resource.Option
	for _, state := range positions {
		opts = append(opts, resource.WithInitialRecord(directionToID(state.Direction), state))
	}
	return WithPositionsOption(opts...)
}

// WithPreset returns an option that configures the model with the given preset.
// This option does not apply to any resource.
func WithPreset(desc *OpenClosePositions_Preset, positions ...*OpenClosePosition) resource.Option {
	sortPositions(positions)
	return modelOptionFunc(func(args *modelArgs) {
		args.presets = append(args.presets, preset{desc: desc, positions: positions})
	})
}

// WithOpenPercentAttributes returns an option that configures the model with the given open percent attributes.
// These attributes are reported via OpenCloseInfo.DescribePositions and inform clients of the bounds and step
// available for the open percent value.
func WithOpenPercentAttributes(attrs *typespb.FloatAttributes) resource.Option {
	return modelOptionFunc(func(args *modelArgs) {
		args.openPercentAttrs = attrs
	})
}

func calcModelArgs(opts ...resource.Option) modelArgs {
	args := new(modelArgs)
	args.apply(DefaultModelOptions...)
	args.apply(opts...)
	return *args
}

type modelArgs struct {
	positionsOpts    []resource.Option
	presets          []preset
	openPercentAttrs *typespb.FloatAttributes
}

func (a *modelArgs) apply(opts ...resource.Option) {
	for _, opt := range opts {
		if v, ok := opt.(modelOption); ok {
			v.applyModel(a)
			continue
		}
		a.positionsOpts = append(a.positionsOpts, opt)
	}
}

func modelOptionFunc(fn func(args *modelArgs)) modelOption {
	return modelOption{resource.EmptyOption{}, fn}
}

type modelOption struct {
	resource.Option
	fn func(args *modelArgs)
}

func (m modelOption) applyModel(args *modelArgs) {
	m.fn(args)
}
