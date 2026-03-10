package fanspeedpb

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

var DefaultPresets = []Preset{
	{Name: "off", Percentage: 0},
	{Name: "low", Percentage: 15},
	{Name: "med", Percentage: 40},
	{Name: "high", Percentage: 75},
	{Name: "full", Percentage: 100},
}

type Preset struct {
	Name       string
	Percentage float32
}

// Model provides a data structure for supporting the Fan Speed trait.
// Unless configured otherwise, this model supports DefaultPresets presets.
type Model struct {
	fanSpeed *resource.Value // of *traits.FanSpeed

	presets []Preset
}

// NewModel constructs a Model with default values using the first of DefaultPresets.
func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		presets:  args.presets,
		fanSpeed: resource.NewValue(args.fanSpeedOpts...),
	}
}

// FanSpeed gets the current fan speed.
func (m *Model) FanSpeed(opts ...resource.ReadOption) *traits.FanSpeed {
	return m.fanSpeed.Get(opts...).(*traits.FanSpeed)
}

// UpdateFanSpeed changes the fan speed.
// Provide write options to support features like relative changes.
// Related properties of the fan speed will be set automatically,
// for example if the preset changes then the percentage and preset index are also changed.
// Providing a resource.InterceptAfter option will disable related property updating.
// Using DeriveValues can re-enable related property computation if needed.
func (m *Model) UpdateFanSpeed(fanSpeed *traits.FanSpeed, opts ...resource.WriteOption) (*traits.FanSpeed, error) {
	if err := m.validateUpdate(fanSpeed); err != nil {
		return nil, err
	}

	opts = append([]resource.WriteOption{resource.InterceptAfter(m.DeriveValues)}, opts...)
	val, err := m.fanSpeed.Set(fanSpeed, opts...)
	if val == nil {
		return nil, err
	}
	return val.(*traits.FanSpeed), err
}

func (m *Model) validateUpdate(fanSpeed *traits.FanSpeed) error {
	if fanSpeed.Preset != "" {
		var found bool
		for _, preset := range m.presets {
			if preset.Name == fanSpeed.Preset {
				found = true
				break
			}
		}
		if !found {
			return status.Errorf(codes.InvalidArgument, "unknown preset %v", fanSpeed.Preset)
		}
	}
	return nil
}

// DeriveValues derives values that haven't been set from those that have.
// For example setting percentage when the preset changes.
// Updated values will be set on new.
func (m *Model) DeriveValues(old, new proto.Message) {
	oldVal := old.(*traits.FanSpeed)
	newVal := new.(*traits.FanSpeed)
	if oldVal.Preset != newVal.Preset {
		// preset updated, keep the index and percentage in sync
		for i, preset := range m.presets {
			if preset.Name == newVal.Preset {
				newVal.PresetIndex = int32(i)
				newVal.Percentage = preset.Percentage
				break
			}
		}
		return
	}

	if oldVal.PresetIndex != newVal.PresetIndex {
		// cap the index if needed, and update the preset and percentage
		if newVal.PresetIndex >= int32(len(m.presets)) {
			newVal.PresetIndex = int32(len(m.presets) - 1)
		}
		if newVal.PresetIndex < 0 {
			newVal.PresetIndex = 0
		}
		preset := m.presets[newVal.PresetIndex]
		newVal.Preset = preset.Name
		newVal.Percentage = preset.Percentage
		return
	}

	if oldVal.Percentage != newVal.Percentage {
		// try to find a preset that matches, and update the index and preset
		newVal.PresetIndex = -1
		newVal.Preset = ""
		for i, preset := range m.presets {
			if preset.Percentage == newVal.Percentage {
				newVal.PresetIndex = int32(i)
				newVal.Preset = preset.Name
				break
			}
		}
		return
	}
}

//goland:noinspection GoNameStartsWithPackageName
type FanSpeedChange struct {
	Value      *traits.FanSpeed
	ChangeTime time.Time
}

func (m *Model) PullFanSpeed(ctx context.Context, opts ...resource.ReadOption) <-chan FanSpeedChange {
	send := make(chan FanSpeedChange)
	go func() {
		defer close(send)
		for change := range m.fanSpeed.Pull(ctx, opts...) {
			val := change.Value.(*traits.FanSpeed)
			select {
			case <-ctx.Done():
				return
			case send <- FanSpeedChange{Value: val, ChangeTime: change.ChangeTime}:
			}
		}
	}()
	return send
}
