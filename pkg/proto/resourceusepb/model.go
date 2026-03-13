package resourceusepb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

// Model stores a single ResourceUse value.
type Model struct {
	value *resource.Value // of *resourceusepb.ResourceUse
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&ResourceUse{})}
	return &Model{
		value: resource.NewValue(append(defaultOptions, opts...)...),
	}
}

func (m *Model) GetResourceUse(opts ...resource.ReadOption) (*ResourceUse, error) {
	return m.value.Get(opts...).(*ResourceUse), nil
}

func (m *Model) SetResourceUse(v *ResourceUse, opts ...resource.WriteOption) (*ResourceUse, error) {
	res, err := m.value.Set(v, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*ResourceUse), nil
}

func (m *Model) PullResourceUse(ctx context.Context, opts ...resource.ReadOption) <-chan PullResourceUseChange {
	return resources.PullValue[*ResourceUse](ctx, m.value.Pull(ctx, opts...))
}

type PullResourceUseChange = resources.ValueChange[*ResourceUse]
