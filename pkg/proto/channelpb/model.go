package channelpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	chosenChannel *resource.Value // of *Channel
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&Channel{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		chosenChannel: resource.NewValue(opts...),
	}
}

func (m *Model) GetChosenChannel(opts ...resource.ReadOption) (*Channel, error) {
	return m.chosenChannel.Get(opts...).(*Channel), nil
}

func (m *Model) UpdateChosenChannel(channel *Channel, opts ...resource.WriteOption) (*Channel, error) {
	v, err := m.chosenChannel.Set(channel, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*Channel), nil
}

func (m *Model) PullChosenChannel(ctx context.Context, opts ...resource.ReadOption) <-chan PullChosenChannelChange {
	send := make(chan PullChosenChannelChange)
	recv := m.chosenChannel.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullChosenChannelChange{
				Value:      change.Value.(*Channel),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullChosenChannelChange struct {
	Value      *Channel
	ChangeTime time.Time
}
