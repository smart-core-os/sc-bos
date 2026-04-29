package speakerpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	volume *resource.Value // of *typespb.AudioLevel
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&typespb.AudioLevel{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		volume: resource.NewValue(opts...),
	}
}

func (m *Model) GetVolume(opts ...resource.ReadOption) (*typespb.AudioLevel, error) {
	return m.volume.Get(opts...).(*typespb.AudioLevel), nil
}

func (m *Model) UpdateVolume(volume *typespb.AudioLevel, opts ...resource.WriteOption) (*typespb.AudioLevel, error) {
	v, err := m.volume.Set(volume, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*typespb.AudioLevel), nil
}

func (m *Model) PullVolume(ctx context.Context, opts ...resource.ReadOption) <-chan PullVolumeChange {
	send := make(chan PullVolumeChange)
	recv := m.volume.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullVolumeChange{
				Value:      change.Value.(*typespb.AudioLevel),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullVolumeChange struct {
	Value      *typespb.AudioLevel
	ChangeTime time.Time
}
