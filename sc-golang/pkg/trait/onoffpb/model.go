package onoffpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	onOff *resource.Value // of *traits.OnOff
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		onOff: resource.NewValue(args.onOffOpts...),
	}
}

func (m *Model) GetOnOff(opts ...resource.ReadOption) (*traits.OnOff, error) {
	return m.onOff.Get(opts...).(*traits.OnOff), nil
}

func (m *Model) UpdateOnOff(value *traits.OnOff, opts ...resource.WriteOption) (*traits.OnOff, error) {
	res, err := m.onOff.Set(value, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.OnOff), nil
}

func (m *Model) PullOnOff(ctx context.Context, opts ...resource.ReadOption) <-chan PullOnOffChange {
	send := make(chan PullOnOffChange)

	recv := m.onOff.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*traits.OnOff)
			send <- PullOnOffChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send
}

type PullOnOffChange struct {
	Value      *traits.OnOff
	ChangeTime time.Time
}
