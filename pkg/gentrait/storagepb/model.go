package storagepb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/storagepb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model stores a single Storage value.
type Model struct {
	value *resource.Value // of *storagepb.Storage
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&storagepb.Storage{})}
	return &Model{
		value: resource.NewValue(append(defaultOptions, opts...)...),
	}
}

func (m *Model) GetStorage(opts ...resource.ReadOption) (*storagepb.Storage, error) {
	return m.value.Get(opts...).(*storagepb.Storage), nil
}

func (m *Model) SetStorage(v *storagepb.Storage, opts ...resource.WriteOption) (*storagepb.Storage, error) {
	res, err := m.value.Set(v, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*storagepb.Storage), nil
}

func (m *Model) PullStorage(ctx context.Context, opts ...resource.ReadOption) <-chan PullStorageChange {
	return resources.PullValue[*storagepb.Storage](ctx, m.value.Pull(ctx, opts...))
}

type PullStorageChange = resources.ValueChange[*storagepb.Storage]
