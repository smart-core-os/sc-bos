package colorpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	color *resource.Value // of *Color
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Color{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		color: resource.NewValue(opts...),
	}
}

func (m *Model) GetColor(opts ...resource.ReadOption) (*Color, error) {
	return m.color.Get(opts...).(*Color), nil
}

func (m *Model) UpdateColor(color *Color, opts ...resource.WriteOption) (*Color, error) {
	v, err := m.color.Set(color, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*Color), nil
}

func (m *Model) PullColor(ctx context.Context, opts ...resource.ReadOption) <-chan PullColorChange {
	send := make(chan PullColorChange)
	recv := m.color.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullColorChange{
				Value:      change.Value.(*Color),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullColorChange struct {
	Value      *Color
	ChangeTime time.Time
}
