package accesspb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

type Model struct {
	accessAttempt *resource.Value // of *accesspb.AccessAttempt
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&AccessAttempt{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		accessAttempt: resource.NewValue(opts...),
	}
}

func (m *Model) GetLastAccessAttempt(opts ...resource.ReadOption) (*AccessAttempt, error) {
	v := m.accessAttempt.Get(opts...)
	return v.(*AccessAttempt), nil
}

func (m *Model) UpdateLastAccessAttempt(accessAttempt *AccessAttempt, opts ...resource.WriteOption) (*AccessAttempt, error) {
	v, err := m.accessAttempt.Set(accessAttempt, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*AccessAttempt), nil
}

func (m *Model) PullAccessAttempts(ctx context.Context, opts ...resource.ReadOption) <-chan PullAccessAttemptsChange {
	return resources.PullValue[*AccessAttempt](ctx, m.accessAttempt.Pull(ctx, opts...))
}

type PullAccessAttemptsChange = resources.ValueChange[*AccessAttempt]
