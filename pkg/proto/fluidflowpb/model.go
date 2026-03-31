package fluidflowpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

type Model struct {
	fluidFlow *resource.Value // of *fluidflowpb.FluidFlow
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&FluidFlow{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		fluidFlow: resource.NewValue(opts...),
	}
}

func (m *Model) GetFluidFlow() (*FluidFlow, error) {
	return m.fluidFlow.Get().(*FluidFlow), nil
}

func (m *Model) UpdateFluidFlow(flow *FluidFlow, opts ...resource.WriteOption) (*FluidFlow, error) {
	res, err := m.fluidFlow.Set(flow, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*FluidFlow), nil
}

func (m *Model) PullFluidFlow(ctx context.Context, opts ...resource.ReadOption) <-chan FlowChange {
	return resources.PullValue[*FluidFlow](ctx, m.fluidFlow.Pull(ctx, opts...))
}

type FlowChange = resources.ValueChange[*FluidFlow]
