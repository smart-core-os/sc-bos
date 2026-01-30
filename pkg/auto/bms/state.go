package bms

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/exp/maps"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/auto/bms/config"
)

func NewReadState() *ReadState {
	return &ReadState{
		Now:            time.Now,
		AirTemperature: make(map[DeviceName]Value[*traits.AirTemperature]),
		Modes:          make(map[DeviceName]map[string]Value[string]),
		Occupancy:      make(map[DeviceName]Value[*traits.Occupancy]),
	}
}

func NewWriteState() *WriteState {
	return &WriteState{
		Now:             time.Now,
		Modes:           make(map[DeviceName]Value[*traits.ModeValues]),
		AirTemperatures: make(map[DeviceName]Value[*traits.AirTemperature]),
	}
}

type DeviceName = string

type ReadState struct {
	Config    config.Root
	Now       func() time.Time
	StartTime time.Time // when was the automation started

	AirTemperature map[DeviceName]Value[*traits.AirTemperature]
	Modes          map[DeviceName]map[string]Value[string]
	Occupancy      map[DeviceName]Value[*traits.Occupancy]

	MeanOATemp *types.Temperature // mean outdoor air temperature
}

func (s *ReadState) Clone() *ReadState {
	clone := NewReadState()
	clone.StartTime = s.StartTime
	// assume config and values in the map are immutable!
	clone.Config = s.Config
	maps.Copy(clone.AirTemperature, s.AirTemperature)
	maps.Copy(clone.Modes, s.Modes)
	maps.Copy(clone.Occupancy, s.Occupancy)
	return clone
}

type WriteState struct {
	Now     func() time.Time // override for testing
	T0, T1  time.Time
	Reasons []string

	Modes           map[DeviceName]Value[*traits.ModeValues]
	AirTemperatures map[DeviceName]Value[*traits.AirTemperature]
}

// Before should be called before processing starts.
func (ws *WriteState) Before() {
	ws.T0 = ws.Now()
	ws.Reasons = ws.Reasons[:0] // save some allocations
}

// After should be called after processing has completed.
func (ws *WriteState) After() {
	ws.T1 = ws.Now()
}

// CopyFromReadState copies the values from rs into ws.
// This keeps the write state up to date with information we've explicitly read, which can then be used to make cache decisions.
func (ws *WriteState) CopyFromReadState(rs *ReadState) {
	propagation := rs.Config.WriteReadPropagation.Or(config.DefaultWriteReadPropagation)

	for name, readVal := range rs.AirTemperature {
		writeVal := ws.AirTemperatures[name]
		if readVal.V != nil && readVal.At.Sub(writeVal.At) > propagation {
			writeVal.V = readVal.V
			writeVal.At = readVal.At
		}
		ws.AirTemperatures[name] = writeVal
	}

	for name, readModes := range rs.Modes {
		writeVal := ws.Modes[name]
		if writeVal.V == nil {
			writeVal.V = &traits.ModeValues{Values: make(map[string]string)}
		}
		for key, readVal := range readModes {
			if readVal.V != "" && readVal.At.Sub(writeVal.At) > propagation {
				writeVal.V.Values[key] = readVal.V
			}
		}
		ws.Modes[name] = writeVal
	}
}

func (ws *WriteState) AddReason(reason string) {
	if reason != "" {
		ws.Reasons = append(ws.Reasons, reason)
	}
}

func (ws *WriteState) AddReasonf(format string, args ...any) {
	ws.AddReason(fmt.Sprintf(format, args...))
}

// Value represents either a value read from a device or a value written to a device.
type Value[V any] struct {
	V   V
	At  time.Time // Read or write time
	Err error     // an error associated with either the read or write
	Hit int       // how many times did we hit the cache, or has the value been updated
}

func (v *Value[V]) set(at time.Time, value V, err error) {
	v.V = value
	v.At = at
	v.Err = err
	v.Hit++
}

func (v *Value[V]) hit() {
	v.Hit++
}

// processPatches applies patches to workingState and emits changes *ReadState.
// Patches will be coalesced and only the latest ReadState will be emitted if receivers on newStateAvailable are slow.
func processPatches(ctx context.Context, workingState *ReadState, changes <-chan Patcher, newStateAvailable chan<- *ReadState) error {
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
		drainChanges := true
		for drainChanges {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				applyChange(change)
			default:
				drainChanges = false
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case change := <-changes:
			applyChange(change)
		case <-readyToNotify:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case newStateAvailable <- workingState:
				// We clone because applying patches happens concurrently with processing any given ReadState.
				// This is desirable to avoid the patch sources from blocking for too long.
				workingState = workingState.Clone()
				readyToNotify = nil
			case change := <-changes:
				applyChange(change)
			}
		}
	}
}
