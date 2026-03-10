package modepb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// DefaultModes define the available modes of a Model if none are provided.
var DefaultModes = &traits.Modes{
	Modes: []*traits.Modes_Mode{
		{Name: "temperature", Ordered: true, Values: []*traits.Modes_Value{
			{Name: "delicates"},
			{Name: "medium"},
			{Name: "whites"},
		}},
		{Name: "spin", Ordered: false, Values: []*traits.Modes_Value{
			{Name: "auto"},
			{Name: "slow"},
			{Name: "fast"},
		}},
	},
}

// Model provides a data structure for supporting the Fan Speed trait.
// Unless configured otherwise, this model supports DefaultModes presets.
type Model struct {
	modeValues *resource.Value // of *traits.ModeValues

	modes *traits.Modes
}

// NewModel is like NewModelModes(DefaultModes).
func NewModel() *Model {
	return NewModelModes(DefaultModes)
}

// NewModelModes constructs a Model with the given modes.
// The first value of each mode will be selected.
func NewModelModes(modes *traits.Modes) *Model {
	modeValues := &traits.ModeValues{
		Values: make(map[string]string),
	}
	for _, mode := range modes.Modes {
		modeValues.Values[mode.Name] = mode.Values[0].Name
	}

	model := &Model{
		modes: DefaultModes,
		modeValues: resource.NewValue(
			resource.WithInitialValue(modeValues),
		),
	}

	return model
}

// ModeValues gets the current mode values.
func (m *Model) ModeValues(opts ...resource.ReadOption) *traits.ModeValues {
	return m.modeValues.Get(opts...).(*traits.ModeValues)
}

func (m *Model) UpdateModeValues(values *traits.ModeValues, opts ...resource.WriteOption) (*traits.ModeValues, error) {
	res, err := m.modeValues.Set(values, opts...)
	if res == nil {
		return nil, err
	}
	return res.(*traits.ModeValues), err
}

//goland:noinspection GoNameStartsWithPackageName
type ModeValuesChange struct {
	Value      *traits.ModeValues
	ChangeTime time.Time
}

func (m *Model) PullModeValues(ctx context.Context, opts ...resource.ReadOption) <-chan ModeValuesChange {
	send := make(chan ModeValuesChange)

	go func() {
		defer close(send)
		for change := range m.modeValues.Pull(ctx, opts...) {
			val := change.Value.(*traits.ModeValues)
			select {
			case <-ctx.Done():
				return
			case send <- ModeValuesChange{Value: val, ChangeTime: change.ChangeTime}:
			}
		}
	}()

	return send
}

func (m *Model) Modes() *traits.Modes {
	return m.modes
}

func (m *Model) AvailableValues(modeName string) []*traits.Modes_Value {
	for _, mode := range m.Modes().Modes {
		if mode.Name == modeName {
			return mode.Values
		}
	}
	return nil
}
