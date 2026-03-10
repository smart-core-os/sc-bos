package buttonpb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	buttonState *resource.Value // of *buttonpb.ButtonState
}

func NewModel(initialPressState ButtonState_Press) *Model {
	return &Model{
		buttonState: resource.NewValue(resource.WithInitialValue(&ButtonState{
			State: initialPressState,
		})),
	}
}

func (m *Model) GetButtonState(options ...resource.ReadOption) *ButtonState {
	return m.buttonState.Get(options...).(*ButtonState)
}

func (m *Model) UpdateButtonState(value *ButtonState, options ...resource.WriteOption) (*ButtonState, error) {
	updated, err := m.buttonState.Set(value, options...)
	if err != nil {
		return nil, err
	}
	return updated.(*ButtonState), nil
}

func (m *Model) PullButtonState(ctx context.Context, options ...resource.ReadOption) <-chan PullButtonStateChange {
	return resources.PullValue[*ButtonState](ctx, m.buttonState.Pull(ctx, options...))
}

type PullButtonStateChange = resources.ValueChange[*ButtonState]
