package resourceusepb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model stores a single ResourceUse value.
type Model struct {
	value *resource.Value // of *resourceusepb.ResourceUse
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&resourceusepb.ResourceUse{})}
	return &Model{
		value: resource.NewValue(append(defaultOptions, opts...)...),
	}
}

func (m *Model) GetResourceUse(opts ...resource.ReadOption) (*resourceusepb.ResourceUse, error) {
	return m.value.Get(opts...).(*resourceusepb.ResourceUse), nil
}

func (m *Model) SetResourceUse(v *resourceusepb.ResourceUse, opts ...resource.WriteOption) (*resourceusepb.ResourceUse, error) {
	res, err := m.value.Set(v, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*resourceusepb.ResourceUse), nil
}

func (m *Model) PullResourceUse(ctx context.Context, opts ...resource.ReadOption) <-chan PullResourceUseChange {
	return resources.PullValue[*resourceusepb.ResourceUse](ctx, m.value.Pull(ctx, opts...))
}

type PullResourceUseChange = resources.ValueChange[*resourceusepb.ResourceUse]
