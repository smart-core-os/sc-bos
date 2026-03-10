package lightpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	brightness *resource.Value
	presets    []preset
}

func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		brightness: resource.NewValue(args.brightnessOpts...),
		presets:    args.presets,
	}
}

func (m *Model) GetBrightness(opts ...resource.ReadOption) (*traits.Brightness, error) {
	return m.brightness.Get(opts...).(*traits.Brightness), nil
}

func (m *Model) UpdateBrightness(light *traits.Brightness, opts ...resource.WriteOption) (*traits.Brightness, error) {
	if m.setLevelFromPreset(light) {
		opts = append(opts, resource.WithMoreUpdatePaths("level_percent"))
	}
	res, err := m.brightness.Set(light, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*traits.Brightness), nil
}

func (m *Model) PullBrightness(ctx context.Context, opts ...resource.ReadOption) <-chan PullBrightnessChange {
	send := make(chan PullBrightnessChange)

	recv := m.brightness.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*traits.Brightness)
			send <- PullBrightnessChange{
				Value:      value,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	return send
}

func (m *Model) ListPresets() []*traits.LightPreset {
	var res []*traits.LightPreset
	for _, p := range m.presets {
		res = append(res, p.LightPreset)
	}
	return res
}

func (m *Model) setLevelFromPreset(b *traits.Brightness) bool {
	if b.GetPreset() == nil {
		return false
	}
	for _, p := range m.presets {
		if p.Name == b.GetPreset().GetName() {
			b.LevelPercent = p.levelPercent
			b.Preset = p.LightPreset // sets the title if needed
			return true
		}
	}
	return false
}

type PullBrightnessChange struct {
	Value      *traits.Brightness
	ChangeTime time.Time
}

type preset struct {
	*traits.LightPreset
	levelPercent float32
}
