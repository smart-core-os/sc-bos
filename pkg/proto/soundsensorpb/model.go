package soundsensorpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type Model struct {
	soundLevel *resource.Value // of *soundsensorpb.SoundLevel
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&SoundLevel{})}
	opts = append(defaultOpts, opts...)

	return &Model{
		soundLevel: resource.NewValue(opts...),
	}
}

func (m *Model) GetSoundLevel(opts ...resource.ReadOption) (*SoundLevel, error) {
	return m.soundLevel.Get(opts...).(*SoundLevel), nil
}

func (m *Model) PullSoundLevel(ctx context.Context, opts ...resource.ReadOption) <-chan PullSoundLevelChange {
	return resources.PullValue[*SoundLevel](ctx, m.soundLevel.Pull(ctx, opts...))
}

func (m *Model) UpdateSoundLevel(soundLevel *SoundLevel, opts ...resource.WriteOption) (*SoundLevel, error) {
	res, err := m.soundLevel.Set(soundLevel, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*SoundLevel), nil
}

type PullSoundLevelChange = resources.ValueChange[*SoundLevel]
