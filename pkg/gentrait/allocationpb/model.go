package allocationpb

import (
	"context"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	mtx        sync.Mutex
	allocation *resource.Value // of *allocationpb.Allocation
}

func NewModel() *Model {
	return &Model{
		allocation: resource.NewValue(resource.WithInitialValue(&allocationpb.Allocation{})),
	}
}

func (m *Model) UpdateAllocation(a *allocationpb.Allocation, opts ...resource.WriteOption) {
	_, _ = m.allocation.Set(a, opts...)
}

func (m *Model) GetAllocation() *allocationpb.Allocation {
	val := m.allocation.Get()
	return val.(*allocationpb.Allocation)
}

func (m *Model) PullAllocation(ctx context.Context, opts ...resource.ReadOption) <-chan PullAllocationChange {
	return resources.PullValue[*allocationpb.Allocation](ctx, m.allocation.Pull(ctx, opts...))
}

type PullAllocationChange = resources.ValueChange[*allocationpb.Allocation]
