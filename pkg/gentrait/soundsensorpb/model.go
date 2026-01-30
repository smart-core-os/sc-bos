package soundsensorpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	soundLevel *resource.Value // of *soundsensorpb.SoundLevel
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&soundsensorpb.SoundLevel{})}
	opts = append(defaultOpts, opts...)

	return &Model{
		soundLevel: resource.NewValue(opts...),
	}
}

func (m *Model) GetSoundLevel(opts ...resource.ReadOption) (*soundsensorpb.SoundLevel, error) {
	return m.soundLevel.Get(opts...).(*soundsensorpb.SoundLevel), nil
}

func (m *Model) PullSoundLevel(ctx context.Context, opts ...resource.ReadOption) <-chan PullSoundLevelChange {
	return resources.PullValue[*soundsensorpb.SoundLevel](ctx, m.soundLevel.Pull(ctx, opts...))
}

func (m *Model) UpdateSoundLevel(soundLevel *soundsensorpb.SoundLevel, opts ...resource.WriteOption) (*soundsensorpb.SoundLevel, error) {
	res, err := m.soundLevel.Set(soundLevel, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*soundsensorpb.SoundLevel), nil
}

type PullSoundLevelChange = resources.ValueChange[*soundsensorpb.SoundLevel]
