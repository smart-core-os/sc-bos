package accesspb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	accessAttempt *resource.Value // of *traits.AccessAttempt
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&traits.AccessAttempt{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		accessAttempt: resource.NewValue(opts...),
	}
}

func (m *Model) GetLastAccessAttempt(opts ...resource.ReadOption) (*traits.AccessAttempt, error) {
	v := m.accessAttempt.Get(opts...)
	return v.(*traits.AccessAttempt), nil
}

func (m *Model) UpdateLastAccessAttempt(accessAttempt *traits.AccessAttempt, opts ...resource.WriteOption) (*traits.AccessAttempt, error) {
	v, err := m.accessAttempt.Set(accessAttempt, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*traits.AccessAttempt), nil
}

func (m *Model) PullAccessAttempts(ctx context.Context, opts ...resource.ReadOption) <-chan PullAccessAttemptsChange {
	send := make(chan PullAccessAttemptsChange)

	recv := m.accessAttempt.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*traits.AccessAttempt)
			send <- PullAccessAttemptsChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

type PullAccessAttemptsChange struct {
	Value      *traits.AccessAttempt
	ChangeTime time.Time
}
