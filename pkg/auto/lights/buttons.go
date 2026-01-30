package lights

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
)

// captureButtonActions consumes button clicks returning the intended actions.
// This will modify writeState to record the last time a button was pressed to avoid an old button press triggering multiple times.
func captureButtonActions(readState *ReadState, writeState *WriteState) (onButtonClicked bool, offButtonClicked bool) {
	mostRecentButtonName, mostRecentButtonState, buttonsFound := getMostRecentButtonPress(readState)
	if !buttonsFound {
		return false, false
	}

	if !isButtonActionRequired(mostRecentButtonState, writeState) {
		writeState.AddReasonf("button action not required")
		return false, false
	}

	// Either onButtonClicked or offButtonClicked should be set
	buttonType := getButtonType(readState, mostRecentButtonName)
	switch buttonType {
	case OnButton:
		writeState.AddReason("on button clicked")
		onButtonClicked = true
	case OffButton:
		writeState.AddReason("off button clicked")
		offButtonClicked = true
	case ToggleButton:
		// decide if the toggle button should be treated as an on or off button based on updates we've written
		if brightnessAllOff(writeState) {
			writeState.AddReason("toggle button clicked")
			writeState.AddReason("all lights off")
			onButtonClicked = true
		} else {
			writeState.AddReason("toggle button clicked")
			writeState.AddReason("some lights are on")
			offButtonClicked = true
		}
	default:
		writeState.AddReason("unknown button type")
		return false, false
	}
	buttonClickTime := mostRecentButtonState.StateChangeTime.AsTime()
	// Update the last time a button action happened
	if onButtonClicked {
		writeState.LastButtonOnTime = buttonClickTime
	}
	writeState.LastButtonAction = buttonClickTime
	return onButtonClicked, offButtonClicked
}

// getButtonType returns the type of button based on where it appeared in the config
func getButtonType(state *ReadState, buttonName deviceName) ButtonType {
	nFound := 0
	buttonType := UndefinedButton
	for _, name := range state.Config.OnButtons {
		if name == buttonName {
			nFound++
			buttonType = OnButton
		}
	}
	for _, name := range state.Config.OffButtons {
		if name == buttonName {
			nFound++
			buttonType = OffButton
		}
	}
	for _, name := range state.Config.ToggleButtons {
		if name == buttonName {
			nFound++
			buttonType = ToggleButton
		}
	}
	// Todo: Add some logging if nButtons != 1
	return buttonType
}

func getMostRecentButtonPress(readState *ReadState) (name deviceName, state *buttonpb.ButtonState, ok bool) {
	mostRecentTime := time.Time{}
	for n, button := range readState.Buttons {
		if button.StateChangeTime.AsTime().After(mostRecentTime) {
			mostRecentTime = button.StateChangeTime.AsTime()
			name = n
			state = button
		}
	}
	return name, state, !mostRecentTime.IsZero()
}

// isButtonActionRequired returns true if state is unpressed and change time is more recent than last button action
func isButtonActionRequired(button *buttonpb.ButtonState, writeState *WriteState) bool {
	if button.State == buttonpb.ButtonState_UNPRESSED && button.StateChangeTime.AsTime().After(writeState.LastButtonAction) {
		return true
	}
	return false
}

type ButtonType int

const (
	UndefinedButton ButtonType = iota
	OnButton
	OffButton
	ToggleButton
)

func (s ButtonType) String() string {
	return [...]string{"Undefined button", "On button", "Off button", "Toggle button"}[s]
}
