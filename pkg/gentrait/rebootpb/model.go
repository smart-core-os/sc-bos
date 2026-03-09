package rebootpb

import (
	"context"

	proto "github.com/smart-core-os/sc-bos/pkg/proto/rebootpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model holds the live RebootState for a device or process.
type Model struct {
	rebootState *resource.Value // of *proto.RebootState
}

// NewModel creates a Model with the given initial state.
func NewModel(initialState *proto.RebootState) *Model {
	return &Model{
		rebootState: resource.NewValue(resource.WithInitialValue(initialState)),
	}
}

func (m *Model) GetRebootState(opts ...resource.ReadOption) *proto.RebootState {
	return m.rebootState.Get(opts...).(*proto.RebootState)
}

func (m *Model) UpdateRebootState(value *proto.RebootState, opts ...resource.WriteOption) (*proto.RebootState, error) {
	updated, err := m.rebootState.Set(value, opts...)
	if err != nil {
		return nil, err
	}
	return updated.(*proto.RebootState), nil
}

func (m *Model) PullRebootState(ctx context.Context, opts ...resource.ReadOption) <-chan PullRebootStateChange {
	return resources.PullValue[*proto.RebootState](ctx, m.rebootState.Pull(ctx, opts...))
}

type PullRebootStateChange = resources.ValueChange[*proto.RebootState]
