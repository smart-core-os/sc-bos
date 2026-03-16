package bootpb

import (
	"context"

	proto "github.com/smart-core-os/sc-bos/pkg/proto/bootpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model holds the live BootState for a device or process.
type Model struct {
	bootState *resource.Value // of *proto.BootState
}

// NewModel creates a Model with the given initial state.
func NewModel(initialState *proto.BootState) *Model {
	return &Model{
		bootState: resource.NewValue(resource.WithInitialValue(initialState)),
	}
}

func (m *Model) GetBootState(opts ...resource.ReadOption) *proto.BootState {
	return m.bootState.Get(opts...).(*proto.BootState)
}

func (m *Model) UpdateBootState(value *proto.BootState, opts ...resource.WriteOption) (*proto.BootState, error) {
	updated, err := m.bootState.Set(value, opts...)
	if err != nil {
		return nil, err
	}
	return updated.(*proto.BootState), nil
}

func (m *Model) PullBootState(ctx context.Context, opts ...resource.ReadOption) <-chan PullBootStateChange {
	return resources.PullValue[*proto.BootState](ctx, m.bootState.Pull(ctx, opts...))
}

type PullBootStateChange = resources.ValueChange[*proto.BootState]
