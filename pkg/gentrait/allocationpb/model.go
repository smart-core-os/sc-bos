package allocationpb

import (
	"context"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	mtx        sync.Mutex
	allocation *resource.Value // of *gen.Allocation
}

func NewModel() *Model {
	return &Model{
		allocation: resource.NewValue(resource.WithInitialValue(&gen.Allocation{})),
	}
}

func (m *Model) UpdateAllocation(a *gen.Allocation, opts ...resource.WriteOption) {
	_, _ = m.allocation.Set(a, opts...)
}

func (m *Model) GetAllocation() *gen.Allocation {
	val := m.allocation.Get()
	return val.(*gen.Allocation)
}

func (m *Model) PullAllocation(ctx context.Context, opts ...resource.ReadOption) <-chan PullAllocationChange {
	return resources.PullValue[*gen.Allocation](ctx, m.allocation.Pull(ctx, opts...))
}

type PullAllocationChange = resources.ValueChange[*gen.Allocation]
