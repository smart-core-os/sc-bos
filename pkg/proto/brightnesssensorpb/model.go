package brightnesssensorpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	ambientBrightness *resource.Value // of traits.AmbientBrightness
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		ambientBrightness: resource.NewValue(args.ambientBrightnessOpts...),
	}
}

func (m *Model) GetAmbientBrightness(opts ...resource.ReadOption) (*AmbientBrightness, error) {
	res := m.ambientBrightness.Get(opts...)
	return res.(*AmbientBrightness), nil
}

func (m *Model) UpdateAmbientBrightness(ambientBrightness *AmbientBrightness, opts ...resource.WriteOption) (*AmbientBrightness, error) {
	res, err := m.ambientBrightness.Set(ambientBrightness, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*AmbientBrightness), nil
}

type PullAmbientBrightnessChange struct {
	Value      *AmbientBrightness
	ChangeTime time.Time
}

func (m *Model) PullAmbientBrightness(ctx context.Context, opts ...resource.ReadOption) <-chan PullAmbientBrightnessChange {
	send := make(chan PullAmbientBrightnessChange)

	recv := m.ambientBrightness.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullAmbientBrightnessChange{
				Value:      change.Value.(*AmbientBrightness),
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}
