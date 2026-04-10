package presspb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	pressedState *resource.Value // of *traits.PressedState
}

func NewModel(initialPressState PressedState_Press) *Model {
	return &Model{
		pressedState: resource.NewValue(resource.WithInitialValue(&PressedState{
			State: initialPressState,
		})),
	}
}

func (m *Model) GetPressedState(options ...resource.ReadOption) *PressedState {
	return m.pressedState.Get(options...).(*PressedState)
}

func (m *Model) UpdatePressedState(value *PressedState, options ...resource.WriteOption) (*PressedState, error) {
	updated, err := m.pressedState.Set(value, options...)
	if err != nil {
		return nil, err
	}
	return updated.(*PressedState), nil
}

func (m *Model) PullPressedState(ctx context.Context, options ...resource.ReadOption) <-chan PullPressedStateChange {
	tx := make(chan PullPressedStateChange)

	rx := m.pressedState.Pull(ctx, options...)
	go func() {
		defer close(tx)
		for change := range rx {
			value := change.Value.(*PressedState)
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
	Value         *PressedState
	ChangeTime    time.Time
	LastSeedValue bool
}
