package dataretentionpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

// Model stores a single DataRetention value.
type Model struct {
	value *resource.Value // of *DataRetention
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&DataRetention{})}
	return &Model{
		value: resource.NewValue(append(defaultOptions, opts...)...),
	}
}

func (m *Model) GetDataRetention(opts ...resource.ReadOption) (*DataRetention, error) {
	return m.value.Get(opts...).(*DataRetention), nil
}

func (m *Model) SetDataRetention(v *DataRetention, opts ...resource.WriteOption) (*DataRetention, error) {
	res, err := m.value.Set(v, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*DataRetention), nil
}

func (m *Model) PullDataRetention(ctx context.Context, opts ...resource.ReadOption) <-chan PullDataRetentionChange {
	return resources.PullValue[*DataRetention](ctx, m.value.Pull(ctx, opts...))
}

type PullDataRetentionChange = resources.ValueChange[*DataRetention]
