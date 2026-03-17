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

// NewModel creates a Model with an optional set of resource.Option.
// By default the BootState is initialised to an empty value.
func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&proto.BootState{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		bootState: resource.NewValue(opts...),
	}
}

func (m *Model) GetBootState(opts ...resource.ReadOption) (*proto.BootState, error) {
	return m.bootState.Get(opts...).(*proto.BootState), nil
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
