package presspb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	pressedState *resource.Value // of *traits.PressedState
}

func NewModel(initialPressState traits.PressedState_Press) *Model {
	return &Model{
		pressedState: resource.NewValue(resource.WithInitialValue(&traits.PressedState{
			State: initialPressState,
		})),
	}
}

func (m *Model) GetPressedState(options ...resource.ReadOption) *traits.PressedState {
	return m.pressedState.Get(options...).(*traits.PressedState)
}

func (m *Model) UpdatePressedState(value *traits.PressedState, options ...resource.WriteOption) (*traits.PressedState, error) {
	updated, err := m.pressedState.Set(value, options...)
	if err != nil {
		return nil, err
	}
	return updated.(*traits.PressedState), nil
}

func (m *Model) PullPressedState(ctx context.Context, options ...resource.ReadOption) <-chan PullPressedStateChange {
	tx := make(chan PullPressedStateChange)

	rx := m.pressedState.Pull(ctx, options...)
	go func() {
		defer close(tx)
		for change := range rx {
			value := change.Value.(*traits.PressedState)
			tx <- PullPressedStateChange{
				Value:         value,
				ChangeTime:    change.ChangeTime,
				LastSeedValue: change.LastSeedValue,
			}
		}
	}()
	return tx
}

type PullPressedStateChange struct {
	Value         *traits.PressedState
	ChangeTime    time.Time
	LastSeedValue bool
}
