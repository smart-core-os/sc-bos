package transportpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type Model struct {
	transport *resource.Value // of *transportpb.Transport
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Transport{})}
	opts = append(defaultOpts, opts...)

	return &Model{
		transport: resource.NewValue(opts...),
	}
}

func (m *Model) GetTransport(opts ...resource.ReadOption) (*Transport, error) {
	return m.transport.Get(opts...).(*Transport), nil
}

func (m *Model) UpdateTransport(transport *Transport, opts ...resource.WriteOption) (*Transport, error) {
	res, err := m.transport.Set(transport, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*Transport), nil
}

func (m *Model) PullTransport(ctx context.Context, opts ...resource.ReadOption) <-chan PullTransportChange {
	return resources.PullValue[*Transport](ctx, m.transport.Pull(ctx, opts...))
}

type PullTransportChange = resources.ValueChange[*Transport]
