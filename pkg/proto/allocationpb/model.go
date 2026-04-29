package allocationpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

type Model struct {
	allocation *resource.Value // of *allocationpb.Allocation
}

func NewModel() *Model {
	return &Model{
		allocation: resource.NewValue(resource.WithInitialValue(&Allocation{})),
	}
}

func (m *Model) UpdateAllocation(a *Allocation, opts ...resource.WriteOption) {
	_, _ = m.allocation.Set(a, opts...)
}

func (m *Model) GetAllocation() *Allocation {
	val := m.allocation.Get()
	return val.(*Allocation)
}

func (m *Model) PullAllocation(ctx context.Context, opts ...resource.ReadOption) <-chan PullAllocationChange {
	return resources.PullValue[*Allocation](ctx, m.allocation.Pull(ctx, opts...))
}

type PullAllocationChange = resources.ValueChange[*Allocation]
