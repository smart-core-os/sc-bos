package lights

import (
	"context"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

type ButtonPatches struct {
	name   deviceName
	client buttonpb.ButtonApiClient
	logger *zap.Logger
}

func (p *ButtonPatches) Subscribe(ctx context.Context, changes chan<- Patcher) error {
	defer func() {
		changes <- clearButtonStatePatcher(p.name)
	}()
	return pull.Changes[Patcher](ctx, p, changes, pull.WithLogger(p.logger.Named("button")))
}

func (p *ButtonPatches) Pull(ctx context.Context, changes chan<- Patcher) error {
	stream, err := p.client.PullButtonState(ctx, &buttonpb.PullButtonStateRequest{Name: p.name})
	if err != nil {
		return err
	}

	for {
		res, err := stream.Recv()
		if err != nil {
			return err
		}
		patcher := pullButtonStatePatcher{
			response: res,
		}
		select {
		case changes <- patcher:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *ButtonPatches) Poll(ctx context.Context, changes chan<- Patcher) error {
	res, err := p.client.GetButtonState(ctx, &buttonpb.GetButtonStateRequest{Name: p.name})
	if err != nil {
		return err
	}
	patcher := getButtonStatePatcher{
		name:        p.name,
		buttonState: res,
	}
	select {
	case changes <- patcher:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type pullButtonStatePatcher struct {
	response *buttonpb.PullButtonStateResponse
}

func (p pullButtonStatePatcher) Patch(state *ReadState) {
	for _, change := range p.response.Changes {
		state.Buttons[change.Name] = change.ButtonState
	}
}

type getButtonStatePatcher struct {
	name        deviceName
	buttonState *buttonpb.ButtonState
}

func (p getButtonStatePatcher) Patch(state *ReadState) {
	state.Buttons[p.name] = p.buttonState
}

type clearButtonStatePatcher string

func (name clearButtonStatePatcher) Patch(state *ReadState) {
	delete(state.Buttons, string(name))
}

// Might want this for decided if I should action button press
// does the button state contain a single click gesture that we haven't processed before?
// func isNewSingleClick(oldState, newState *buttonpb.ButtonState) (t time.Time, ok bool) {
// 	oldGesture := oldState.GetMostRecentGesture()
// 	newGesture := newState.GetMostRecentGesture()
//
// 	if newGesture == nil {
// 		return time.Time{}, false
// 	}
// 	hasNewID := oldGesture.GetId() != newGesture.GetId()
// 	isSingleClick := newGesture.Kind == buttonpb.ButtonState_Gesture_CLICK && newGesture.Count == 1
// 	if hasNewID && isSingleClick {
// 		ok = true
// 		t = newGesture.GetEndTime().AsTime()
// 	}
// 	return
// }
