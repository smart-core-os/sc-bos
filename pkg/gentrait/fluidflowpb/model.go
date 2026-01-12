package fluidflowpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/fluidflowpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	fluidFlow *resource.Value // of *fluidflowpb.FluidFlow
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&fluidflowpb.FluidFlow{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		fluidFlow: resource.NewValue(opts...),
	}
}

func (m *Model) GetFluidFlow() (*fluidflowpb.FluidFlow, error) {
	return m.fluidFlow.Get().(*fluidflowpb.FluidFlow), nil
}

func (m *Model) UpdateFluidFlow(flow *fluidflowpb.FluidFlow, opts ...resource.WriteOption) (*fluidflowpb.FluidFlow, error) {
	res, err := m.fluidFlow.Set(flow, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*fluidflowpb.FluidFlow), nil
}

func (m *Model) PullFluidFlow(ctx context.Context, opts ...resource.ReadOption) <-chan FlowChange {
	return resources.PullValue[*fluidflowpb.FluidFlow](ctx, m.fluidFlow.Pull(ctx, opts...))
}

type FlowChange = resources.ValueChange[*fluidflowpb.FluidFlow]
