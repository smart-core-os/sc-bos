package lights

import (
	"context"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/buttonpb"
)

// deviceName is a smart core name for a device.
// Use deviceName instead of string to help with documenting the intent behind the string.
type deviceName = string

// ReadState models everything we have read from the system.
// For example if we PullBrightness, then the responses will be recoded here.
type ReadState struct {
	Config config.Root

	AutoStartTime time.Time // time that the automation started up
	Occupancy     map[deviceName]*traits.Occupancy
	// used for daylight dimming
	AmbientBrightness map[deviceName]*traits.AmbientBrightness
	Buttons           map[deviceName]*buttonpb.ButtonState
	// used for selecting the run modes, aka "modes" config property
	Modes *traits.ModeValues
}

func NewReadState(t time.Time) *ReadState {
	return &ReadState{
		AutoStartTime:     t,
		Occupancy:         make(map[deviceName]*traits.Occupancy),
		AmbientBrightness: make(map[deviceName]*traits.AmbientBrightness),
		Buttons:           make(map[deviceName]*buttonpb.ButtonState),
	}
}

func (s *ReadState) Clone() *ReadState {
	clone := NewReadState(s.AutoStartTime)
	clone.Config = s.Config
	// assume values in the map are immutable!
	for name, val := range s.Occupancy {
		clone.Occupancy[name] = val
	}
	for name, val := range s.AmbientBrightness {
		clone.AmbientBrightness[name] = val
	}
	for name, val := range s.Buttons {
		clone.Buttons[name] = val
	}
	clone.Modes = s.Modes
	return clone
}

func (s *ReadState) Now() time.Time {
	if s.Config.Now == nil {
		return time.Now()
	}
	return s.Config.Now()
}

// ChangesSince returns a human/log compatible list of changes between other and s.
func (s *ReadState) ChangesSince(other *ReadState) []string {
	var changes []string
	changes = append(changes, mapChanges("occupancy", other.Occupancy, s.Occupancy, func(a, b *traits.Occupancy) string {
		if a.State != b.State {
			return fmt.Sprintf("%s->%s", a.State, b.State)
		}
		// we could report on state change time, but that could get noisy
		return ""
	})...)
	changes = append(changes, mapChanges("ambient brightness", other.AmbientBrightness, s.AmbientBrightness, func(a, b *traits.AmbientBrightness) string {
		if math.Abs(float64(a.BrightnessLux-b.BrightnessLux)) > 0.01 {
			return fmt.Sprintf("%.2f->%.2f", a.BrightnessLux, b.BrightnessLux)
		}
		return ""
	})...)
	changes = append(changes, mapChanges("buttons", other.Buttons, s.Buttons, func(a, b *buttonpb.ButtonState) string {
		var changes []string
		if a.State != b.State {
			changes = append(changes, fmt.Sprintf("%s->%s", a.State, b.State))
		}
		if a.MostRecentGesture != b.MostRecentGesture {
			changes = append(changes, fmt.Sprintf("%s->%s", a.MostRecentGesture, b.MostRecentGesture))
		}
		return ""
	})...)
	return changes
}

func mapChanges[V any](prefix string, a, b map[deviceName]V, cmp func(a, b V) string) []string {
	aNotB := map[deviceName]struct{}{}
	bNotA := map[deviceName]struct{}{}
	aAndB := map[deviceName]struct{}{}
	for k := range a {
		aNotB[k] = struct{}{}
	}
	for k := range b {
		if _, ok := a[k]; ok {
			aAndB[k] = struct{}{}
			delete(aNotB, k)
		} else {
			bNotA[k] = struct{}{}
		}
	}

	changes := make([]string, 0, len(aNotB)+len(bNotA)) // at least this many

	for _, k := range slices.Sorted(maps.Keys(aNotB)) {
		changes = append(changes, fmt.Sprintf("%s %s: removed", prefix, k))
	}
	for _, k := range slices.Sorted(maps.Keys(bNotA)) {
		changes = append(changes, fmt.Sprintf("%s %s: added", prefix, k))
	}
	for _, k := range slices.Sorted(maps.Keys(aAndB)) {
		diff := cmp(a[k], b[k])
		if diff != "" {
			changes = append(changes, fmt.Sprintf("%s %s: %s", prefix, k, diff))
		}
	}

	return changes
}

// WriteState models the state of the system based on the changes we've made to it.
// For example if we UpdateBrightness, then the response to that call is recorded in this state.
type WriteState struct {
	Reasons []string

	Brightness       map[deviceName]Value[*traits.Brightness]
	LastButtonAction time.Time // used for button press deduplication, the last time we did anything due to a button press
	LastButtonOnTime time.Time // used for occupancy related darkness, the last time lights were turned on due to button press
	ActiveMode       string
}

// Value is a nuget of data we know about.
type Value[V any] struct {
	V   V
	At  time.Time
	Err error
	Hit int // cache hits
}

func (v *Value[V]) set(at time.Time, value V) {
	v.V = value
	v.At = at
	v.Err = nil
	v.Hit = 0
}

func (v *Value[V]) hit() {
	v.Hit++
}

func NewWriteState(startTime time.Time) *WriteState {
	return &WriteState{
		Brightness: make(map[deviceName]Value[*traits.Brightness]),
		// This causes all button presses before we boot to be ignored for action purposes - i.e. they don't directly turn lights on or off.
		// This doesn't affect occupancy timeouts, so if a button was pressed 2 mins ago it still counts towards unoccupied darkness.
		LastButtonAction: startTime,
	}
}

func (s *WriteState) MergeFrom(other *WriteState) {
	for name, brightness := range other.Brightness {
		if brightness.V == nil {
			delete(s.Brightness, name)
		} else {
			s.Brightness[name] = brightness
		}
	}
	s.LastButtonAction = other.LastButtonAction
}

// Before sets up the write state ready for processing.
func (s *WriteState) Before() {
	s.Reasons = s.Reasons[:0] // save some allocations
}

// After should be called immediately after processing has completed.
func (s *WriteState) After() {
	// nothing to do here yet
}

func (s *WriteState) AddReason(reason string) {
	if reason != "" {
		s.Reasons = append(s.Reasons, reason)
	}
}

func (s *WriteState) AddReasonf(format string, args ...any) {
	s.AddReason(fmt.Sprintf(format, args...))
}

// readStateChanges collates changes and emits *ReadState.
func readStateChanges(ctx context.Context, workingState *ReadState, changes <-chan Patcher, newStateAvailable chan<- *ReadState) error {
	var readyToNotify chan struct{}

	// applyChange updates workingState by applying change to it.
	// This signals that we have some new state ready for someone to process.
	applyChange := func(change Patcher) {
		change.Patch(workingState)

		// let the loop know we have something to broadcast
		if readyToNotify == nil {
			readyToNotify = make(chan struct{})
			close(readyToNotify) // notify without sending anything
		}
	}

	// The following code can be summarised as:
	//   Apply changes to the state, notify when the state has updated
	//
	// As with most things it's a little more complicated than that though.
	// We _really_ want to prioritise updating the state over notifying that the state has updated,
	// if the thing that is processing the state misses some state updates then this isn't a bad thing.
	//
	// To accomplish this we make sure that at each step that could block we're checking for state changes again,
	// in practice this means each select has a case that checks for changes.
	// Unfortunately this doesn't guarantee that changes will be processed before the state is processed because
	// Go doesn't provide any guarantees or mechanisms for prioritising select cases.
	// We try our best though by introducing a 'drain loop' that will empty the changes chan of all items before
	// we consider even looking to see if we should be notifying of state changes.
	for {
	drainChanges:
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				applyChange(change)
			default:
				break drainChanges
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case change := <-changes:
			applyChange(change)
		case <-readyToNotify:
			clonedState := workingState.Clone()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case newStateAvailable <- clonedState:
				readyToNotify = nil
			case change := <-changes:
				applyChange(change)
			}
		}
	}
}
