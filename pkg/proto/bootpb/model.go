package bootpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

// Model holds the live BootState for a device or process.
type Model struct {
	bootState *resource.Value // of *proto.BootState
}

// NewModel creates a Model with an optional set of resource.Option.
// By default the BootState is initialised to an empty value.
func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&BootState{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		bootState: resource.NewValue(opts...),
	}
}

func (m *Model) GetBootState(opts ...resource.ReadOption) (*BootState, error) {
	return m.bootState.Get(opts...).(*BootState), nil
}

func (m *Model) UpdateBootState(value *BootState, opts ...resource.WriteOption) (*BootState, error) {
	updated, err := m.bootState.Set(value, opts...)
	if err != nil {
		return nil, err
	}
	return updated.(*BootState), nil
}

func (m *Model) PullBootState(ctx context.Context, opts ...resource.ReadOption) <-chan PullBootStateChange {
	return resources.PullValue[*BootState](ctx, m.bootState.Pull(ctx, opts...))
}

type PullBootStateChange = resources.ValueChange[*BootState]
