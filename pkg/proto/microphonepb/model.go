package microphonepb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	gain *resource.Value // of *typespb.AudioLevel
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&typespb.AudioLevel{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		gain: resource.NewValue(opts...),
	}
}

func (m *Model) GetGain(opts ...resource.ReadOption) (*typespb.AudioLevel, error) {
	return m.gain.Get(opts...).(*typespb.AudioLevel), nil
}

func (m *Model) UpdateGain(gain *typespb.AudioLevel, opts ...resource.WriteOption) (*typespb.AudioLevel, error) {
	v, err := m.gain.Set(gain, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*typespb.AudioLevel), nil
}

func (m *Model) PullGain(ctx context.Context, opts ...resource.ReadOption) <-chan PullGainChange {
	send := make(chan PullGainChange)
	recv := m.gain.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullGainChange{
				Value:      change.Value.(*typespb.AudioLevel),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullGainChange struct {
	Value      *typespb.AudioLevel
	ChangeTime time.Time
}
