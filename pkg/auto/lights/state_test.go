package lights

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
)

func TestChangesSince_ButtonStateChange(t *testing.T) {
	before := NewReadState(time.Time{})
	before.Buttons["btn1"] = &buttonpb.ButtonState{State: buttonpb.ButtonState_UNPRESSED}

	after := NewReadState(time.Time{})
	after.Buttons["btn1"] = &buttonpb.ButtonState{State: buttonpb.ButtonState_PRESSED}

	changes := after.ChangesSince(before)
	if len(changes) == 0 {
		t.Fatal("expected changes for button state transition, got none")
	}
	found := false
	for _, c := range changes {
		if c != "" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected non-empty change description, got %v", changes)
	}
}

func TestChangesSince_ButtonGestureChange(t *testing.T) {
	before := NewReadState(time.Time{})
	before.Buttons["btn1"] = &buttonpb.ButtonState{
		MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_CLICK},
	}

	after := NewReadState(time.Time{})
	after.Buttons["btn1"] = &buttonpb.ButtonState{
		MostRecentGesture: &buttonpb.ButtonState_Gesture{Kind: buttonpb.ButtonState_Gesture_HOLD},
	}

	changes := after.ChangesSince(before)
	if len(changes) == 0 {
		t.Fatal("expected changes for button gesture transition, got none")
	}
	found := false
	for _, c := range changes {
		if c != "" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected non-empty change description, got %v", changes)
	}
}

func TestChangesSince_NoButtonChange(t *testing.T) {
	state := NewReadState(time.Time{})
	state.Buttons["btn1"] = &buttonpb.ButtonState{State: buttonpb.ButtonState_PRESSED}

	changes := state.ChangesSince(state)
	if len(changes) != 0 {
		t.Errorf("expected no changes when state is identical, got %v", changes)
	}
}
