package accesspb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	accessAttempt *resource.Value // of *accesspb.AccessAttempt
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&accesspb.AccessAttempt{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		accessAttempt: resource.NewValue(opts...),
	}
}

func (m *Model) GetLastAccessAttempt(opts ...resource.ReadOption) (*accesspb.AccessAttempt, error) {
	v := m.accessAttempt.Get(opts...)
	return v.(*accesspb.AccessAttempt), nil
}

func (m *Model) UpdateLastAccessAttempt(accessAttempt *accesspb.AccessAttempt, opts ...resource.WriteOption) (*accesspb.AccessAttempt, error) {
	v, err := m.accessAttempt.Set(accessAttempt, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*accesspb.AccessAttempt), nil
}

func (m *Model) PullAccessAttempts(ctx context.Context, opts ...resource.ReadOption) <-chan PullAccessAttemptsChange {
	return resources.PullValue[*accesspb.AccessAttempt](ctx, m.accessAttempt.Pull(ctx, opts...))
}

type PullAccessAttemptsChange = resources.ValueChange[*accesspb.AccessAttempt]
