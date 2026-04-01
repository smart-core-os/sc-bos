package countpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	count *resource.Value // of *Count
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Count{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		count: resource.NewValue(opts...),
	}
}

func (m *Model) GetCount(opts ...resource.ReadOption) (*Count, error) {
	return m.count.Get(opts...).(*Count), nil
}

func (m *Model) UpdateCount(count *Count, opts ...resource.WriteOption) (*Count, error) {
	v, err := m.count.Set(count, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*Count), nil
}

func (m *Model) PullCounts(ctx context.Context, opts ...resource.ReadOption) <-chan PullCountsChange {
	send := make(chan PullCountsChange)
	recv := m.count.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullCountsChange{
				Value:      change.Value.(*Count),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullCountsChange struct {
	Value      *Count
	ChangeTime time.Time
}
