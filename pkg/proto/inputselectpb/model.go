package inputselectpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	input *resource.Value // of *Input
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Input{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		input: resource.NewValue(opts...),
	}
}

func (m *Model) GetInput(opts ...resource.ReadOption) (*Input, error) {
	return m.input.Get(opts...).(*Input), nil
}

func (m *Model) UpdateInput(input *Input, opts ...resource.WriteOption) (*Input, error) {
	v, err := m.input.Set(input, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*Input), nil
}

func (m *Model) PullInput(ctx context.Context, opts ...resource.ReadOption) <-chan PullInputChange {
	send := make(chan PullInputChange)
	recv := m.input.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullInputChange{
				Value:      change.Value.(*Input),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullInputChange struct {
	Value      *Input
	ChangeTime time.Time
}
