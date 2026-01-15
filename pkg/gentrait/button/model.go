package button

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	buttonState *resource.Value // of *buttonpb.ButtonState
}

func NewModel(initialPressState buttonpb.ButtonState_Press) *Model {
	return &Model{
		buttonState: resource.NewValue(resource.WithInitialValue(&buttonpb.ButtonState{
			State: initialPressState,
		})),
	}
}

func (m *Model) GetButtonState(options ...resource.ReadOption) *buttonpb.ButtonState {
	return m.buttonState.Get(options...).(*buttonpb.ButtonState)
}

func (m *Model) UpdateButtonState(value *buttonpb.ButtonState, options ...resource.WriteOption) (*buttonpb.ButtonState, error) {
	updated, err := m.buttonState.Set(value, options...)
	if err != nil {
		return nil, err
	}
	return updated.(*buttonpb.ButtonState), nil
}

func (m *Model) PullButtonState(ctx context.Context, options ...resource.ReadOption) <-chan PullButtonStateChange {
	return resources.PullValue[*buttonpb.ButtonState](ctx, m.buttonState.Pull(ctx, options...))
}

type PullButtonStateChange = resources.ValueChange[*buttonpb.ButtonState]
