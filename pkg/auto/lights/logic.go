package lights

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
)

// processState executes actions based on both read and write states.
// Here is where the logic that says "when PIRs report occupied, turn lights on" lives.
//
// Returning a non-zero duration indicates that processing should be rerun after this delay even if ReadState doesn't
// change.
func processState(ctx context.Context, readState *ReadState, writeState *WriteState, actions actions) (time.Duration, error) {
	now := readState.Now()
	switchOn, switchOff, mode, rerunAfter := decideAction(now, readState, writeState)

	// ActiveMode is the mode which was most recently asserted on the lights
	// this is used in the logic next time to detect when a mode has changed.
	// We also track when the auto is dis/enabled so we can reassert the light levels on change
	if switchOn || switchOff || mode.DisableAuto {
		writeState.ActiveMode = mode.Name
	}
	if mode.DisableAuto {
		return rerunAfter, nil
	}

	if switchOn {
		level, ok := computeOnLevelPercent(mode, readState, writeState)
		// logger.Debug("Setting level.", zap.Float32("level", level))
		if !ok {
			writeState.AddReason("daylight dimming lacks data")
			// todo: here we are in a position where daylight dimming is supposed to be enabled but we don't have enough
			//  info to actually choose the output light level. We should probably not make any changes and wait for
			//  more data to come in, but we'll leave that to future us as part of snagging.
		}
		return rerunAfter, updateBrightnessLevel(ctx, now, writeState, actions, level, readState.Config.Lights...)
	} else if switchOff {
		offLevel := computeOffLevelPercent(mode)
		return rerunAfter, updateBrightnessLevel(ctx, now, writeState, actions, offLevel, readState.Config.Lights...)
	} else {
		writeState.AddReason("refreshing")
		return rerunAfter, refreshBrightnessLevel(ctx, now, writeState, actions, readState.Config.Lights...)
	}
}

func decideAction(now time.Time, readState *ReadState, writeState *WriteState) (switchOn, switchOff bool, mode config.ModeOption, rerunAfter time.Duration) {
	// Work out what we need to do to apply the given writeState and make those changes for as long as ctx is valid
	mode, rerunAfter = activeMode(now, readState)
	writeState.AddReasonf("mode:%s", mode.Name)
	if mode.DisableAuto {
		writeState.AddReason("disabled")
		return false, false, mode, 0
	}

	// determine if we had already asserted a mode, and then changed to a different one
	// if no brightness was asserted yet, then we don't set modeChanged as that would force a reassertion which we don't want
	var modeChanged bool
	if anyLightsAsserted(writeState) && writeState.ActiveMode != mode.Name {
		modeChanged = true
	}

	onButtonClicked, offButtonClicked := captureButtonActions(readState, writeState)

	if offButtonClicked {
		writeState.AddReason("button off")
		switchOff = true
		return
	}

	anyOccupied := areAnyOccupied(readState.Config.OccupancySensors, readState.Occupancy)

	// We can do easy checks for occupancy and turn things on if they are occupied
	if anyOccupied || onButtonClicked {
		if onButtonClicked {
			if wake := mode.UnoccupiedOffDelay.Duration - now.Sub(writeState.LastButtonOnTime); rerunAfter == 0 || wake < rerunAfter {
				rerunAfter = wake
			}
		}
		if anyOccupied {
			writeState.AddReason("occupied")
		} else {
			writeState.AddReason("button on")
		}
		switchOn = true
		return
	}

	// This code check when occupancy last reported unoccupied and only turns the lights off
	// if it's been unoccupied for more than unoccupied timeout.
	// If it hasn't been 10 minutes yet, it waits some time and turns the lights off when it has been
	// greater than the unoccupied timeout.
	// If a push button hasn't been pressed for the timeout period lights will be switched off too

	becameUnoccupied := lastUnoccupiedTime(readState)
	if buttonOnTime := writeState.LastButtonOnTime; buttonOnTime.After(becameUnoccupied) {
		becameUnoccupied = buttonOnTime
	}
	if becameUnoccupied.IsZero() {
		// we don't know when the lights were last switched on, but we know it must have been before the automation
		// started, so we can use this time
		becameUnoccupied = readState.AutoStartTime
		writeState.AddReason("no occupancy signals, using auto start time")
	}
	if becameUnoccupied.After(now) {
		writeState.AddReason("last recorded occupancy time is in the future")
	}

	sinceUnoccupied := now.Sub(becameUnoccupied)
	unoccupiedDelayBeforeDarkness := mode.UnoccupiedOffDelay.Duration

	if sinceUnoccupied >= unoccupiedDelayBeforeDarkness {
		// we've been unoccupied for long enough, turn things off now
		writeState.AddReasonf("unoccupied for %v", formatDuration(sinceUnoccupied))
		switchOff = true
		return
	} else {
		writeState.AddReasonf("unoccupied %v<%v ago", formatDuration(sinceUnoccupied), formatDuration(unoccupiedDelayBeforeDarkness))
		// when the mode changes, we always want to re-assert the lighting state, so that the new mode will
		// apply
		if modeChanged {
			writeState.AddReason("mode changed")
			switchOn = true
		}

		// we haven't written anything, but in `unoccupiedDelayBeforeDarkness - sinceUnoccupied` time we will, let the
		// caller know
		if wait := unoccupiedDelayBeforeDarkness - sinceUnoccupied; rerunAfter == 0 || wait < rerunAfter {
			rerunAfter = wait
		}
	}

	return
}

const (
	ModeAuto     = "auto"
	ModeDefault  = "default"
	ModeValueKey = "lighting.mode"
)

// activeMode returns the current active mode for the automation, plus the ttl for when that mode is likely to change.
// The active mode is the next mode to stop, or the default mode if no modes are started.
func activeMode(now time.Time, state *ReadState) (config.ModeOption, time.Duration) {
	// check if there's a mode set from the read state
	if mode, ok := readStateMode(state); ok {
		return mode, 0
	}

	var nextStart, nextEnd time.Time
	var currentMode config.ModeOption
	found := false
	for _, mode := range state.Config.Modes {
		if mode.Start == nil || mode.Start.Schedule == nil ||
			mode.End == nil || mode.End.Schedule == nil {
			continue
		}
		startAt := mode.Start.Next(now)
		endAt := mode.End.Next(now)
		if startAt.Before(endAt) {
			// currently stopped
			if nextStart.IsZero() || startAt.Before(nextStart) {
				nextStart = startAt
			}
		} else {
			// currently started
			if nextEnd.IsZero() || endAt.Before(nextEnd) {
				nextEnd = endAt
				currentMode = mode
				found = true
			}
		}
	}

	if found {
		wake := nextStart
		if wake.IsZero() || nextEnd.Before(wake) {
			wake = nextEnd
		}
		return currentMode, wake.Sub(now)
	}

	wake := now
	if nextStart.After(wake) {
		wake = nextStart
	}
	return config.ModeOption{Name: ModeDefault, Mode: state.Config.Mode}, wake.Sub(now)
}

func readStateMode(state *ReadState) (config.ModeOption, bool) {
	if state.Modes == nil {
		return config.ModeOption{}, false
	}
	values := state.Modes.Values
	key := state.Config.ModeValueKey
	if key == "" {
		key = ModeValueKey
	}
	modeName, ok := values[key]
	if !ok {
		return config.ModeOption{}, false
	}
	switch modeName {
	case ModeAuto:
		return config.ModeOption{}, false
	case ModeDefault:
		return config.ModeOption{Name: ModeDefault, Mode: state.Config.Mode}, true
	default:
		for _, mode := range state.Config.Modes {
			if mode.Name == modeName {
				return mode, true
			}
		}
	}
	return config.ModeOption{}, false
}

// brightnessAllOff returns if all the given brightness levels are zero.
// Note is len(brightness) == 0, this will return true.
func brightnessAllOff(state *WriteState) bool {
	for _, brightness := range state.Brightness {
		if brightness.V.GetLevelPercent() > 0 {
			return false
		}
	}
	return true
}

// areAnyOccupied returns true if any occupancy sensors in the list are occupied
func areAnyOccupied(sensorsPresent []deviceName, occupancyStates map[deviceName]*traits.Occupancy) bool {
	var ret = false
	for _, name := range sensorsPresent {
		if o, ok := occupancyStates[name]; ok {
			if o.State == traits.Occupancy_OCCUPIED {
				ret = true
				break
			}
		}
	}
	return ret
}

// lastUnoccupiedTime returns the most recent Occupancy.StateChangeTime across each Config.OccupancySensors that have an
// unoccupied state.
func lastUnoccupiedTime(state *ReadState) time.Time {
	var mostRecentUnoccupiedTime time.Time
	for _, name := range state.Config.OccupancySensors {
		o, ok := state.Occupancy[name]
		if !ok {
			continue
		}

		if o.State == traits.Occupancy_UNOCCUPIED {
			if o.StateChangeTime == nil {
				continue
			}
			candidate := o.StateChangeTime.AsTime()
			if candidate.After(mostRecentUnoccupiedTime) {
				mostRecentUnoccupiedTime = candidate
			}
		}
	}
	return mostRecentUnoccupiedTime
}

func computeOffLevelPercent(mode config.ModeOption) (level float32) {
	if mode.OffLevelPercent != nil {
		return *mode.OffLevelPercent
	}
	return 0
}

func computeOnLevelPercent(mode config.ModeOption, readState *ReadState, writeState *WriteState) (level float32, ok bool) {
	var fullyOff, fullyOn float32 = 0, 100.0
	if mode.OnLevelPercent != nil {
		fullyOn = *mode.OnLevelPercent
	}
	if mode.OffLevelPercent != nil {
		fullyOff = *mode.OffLevelPercent
	}

	dd := mode.DaylightDimming
	if dd == nil {
		return fullyOn, true
	}
	if len(dd.Thresholds) == 0 {
		return fullyOn, true
	}
	if len(readState.AmbientBrightness) == 0 {
		return fullyOn, false
	}

	sensorLux := combinedLuxLevel(readState.AmbientBrightness)
	threshold, ok := closestThresholdBelow(sensorLux, dd.Thresholds)
	if !ok {
		// measured lux level is brighter than the config for the dimmest on level, so just turn the light off
		return fullyOff, true
	}

	// Go half way between goal and current level percent
	currentAverage, err := getAverageLevel(writeState)
	var levelPercent float32
	pcTowardsGoal := mode.DaylightDimming.PercentageTowardsGoal

	if pcTowardsGoal <= 0 || pcTowardsGoal > 100 {
		pcTowardsGoal = 75
	}

	if err == nil {
		levelPercent = currentAverage + pcTowardsGoal*(threshold.LevelPercent-currentAverage)/100
	} else {
		levelPercent = threshold.LevelPercent
	}

	return levelPercent, true
}

func getAverageLevel(state *WriteState) (float32, error) {
	sum := float32(0)
	n := 0
	for _, brightness := range state.Brightness {
		if brightness.V == nil {
			continue
		}
		sum += brightness.V.LevelPercent
		n++
	}
	if n == 0 {
		return 0, errors.New("No brightness readings available")
	}
	return sum / float32(n), nil
}

func combinedLuxLevel(brightness map[deviceName]*traits.AmbientBrightness) float32 {
	var n, v float32
	n, v = float32(len(brightness)), 0
	for _, ambientBrightness := range brightness {
		v += ambientBrightness.BrightnessLux / n
	}
	return v
}

func closestThresholdBelow(lux float32, thresholds []config.LevelThreshold) (config.LevelThreshold, bool) {
	if len(thresholds) == 0 {
		return config.LevelThreshold{}, false
	}

	// BelowLux 100 now comes before 400 in the slice
	sort.Slice(thresholds, func(i, j int) bool {
		return thresholds[i].BelowLux < thresholds[j].BelowLux
	})

	for _, threshold := range thresholds {
		if lux < threshold.BelowLux {
			return threshold, true
		}
	}

	if thresholds[0].BelowLux == 0 {
		return thresholds[0], true
	}
	return config.LevelThreshold{}, false
}

func anyLightsAsserted(writeState *WriteState) bool {
	return len(writeState.Brightness) > 0
}
