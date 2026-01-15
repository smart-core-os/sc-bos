package transport

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	transport *resource.Value // of *transportpb.Transport
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&transportpb.Transport{})}
	opts = append(defaultOpts, opts...)

	return &Model{
		transport: resource.NewValue(opts...),
	}
}

func (m *Model) GetTransport(opts ...resource.ReadOption) (*transportpb.Transport, error) {
	return m.transport.Get(opts...).(*transportpb.Transport), nil
}

func (m *Model) UpdateTransport(transport *transportpb.Transport, opts ...resource.WriteOption) (*transportpb.Transport, error) {
	res, err := m.transport.Set(transport, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*transportpb.Transport), nil
}

func (m *Model) PullTransport(ctx context.Context, opts ...resource.ReadOption) <-chan PullTransportChange {
	return resources.PullValue[*transportpb.Transport](ctx, m.transport.Pull(ctx, opts...))
}

type PullTransportChange = resources.ValueChange[*transportpb.Transport]
