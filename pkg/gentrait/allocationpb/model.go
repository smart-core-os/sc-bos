package allocationpb

import (
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	mtx         sync.Mutex
	allocations map[string]*resource.Value // of *gen.Allocation
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) ListAllocations() *gen.ListAllocatableResourcesResponse {
	var results []*gen.Allocation

	m.mtx.Lock()
	defer m.mtx.Unlock()
	for _, v := range m.allocations {
		results = append(results, v.Get().(*gen.Allocation))
	}
	return &gen.ListAllocatableResourcesResponse{
		Allocations: results,
	}
}

func (m *Model) ListAllocationResources() []*resource.Value {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	var results []*resource.Value
	for _, v := range m.allocations {
		results = append(results, v)
	}
	return results
}

func (m *Model) UpdateAllocation(a *gen.Allocation, opts ...resource.WriteOption) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	value, exists := m.allocations[a.GetId()]
	if !exists {
		value = resource.NewValue(resource.WithInitialValue(&gen.Allocation{}))
		m.allocations[a.GetId()] = value
	}
	_, _ = value.Set(a, opts...)
}
