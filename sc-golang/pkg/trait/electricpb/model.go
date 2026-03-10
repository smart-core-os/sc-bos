package electricpb

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/time/clock"
)

var (
	ErrModeNotFound     = status.Error(codes.NotFound, "electric mode not found")
	ErrNormalModeExists = status.Error(codes.AlreadyExists, "a normal electric mode already exists")
	ErrDeleteActiveMode = status.Error(codes.FailedPrecondition, "attempt to delete active mode")
)

// Model is a simple data store for electric devices. It simply stores the data given to it, and does not implement
// any business logic.
// For the implementation of the gRPC trait based on Model, see ModelServer.
// Invariants:
//  1. At most one mode has normal = true.
//  2. The active mode cannot be deleted.
//  3. Only a mode that exists can be active (except when the Model is first created, when a dummy mode is active)
type Model struct {
	demand     *resource.Value // of *traits.ElectricDemand
	activeMode *resource.Value // of *traits.ElectricMode
	modes      *resource.Collection

	// mu protects invariants
	mu    sync.RWMutex
	clock clock.Clock
	Rng   *rand.Rand // for generating mode ids
}

// NewModel constructs a Model with default values:
//
//	Current: 0
//	Voltage: 240
//	Rating: 13
//
// No modes, active mode is empty.
func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	mem := &Model{
		demand:     resource.NewValue(args.demandOpts...),
		activeMode: resource.NewValue(args.activeModeOpts...),
		modes:      resource.NewCollection(args.modeOpts...),
		clock:      args.clock,
		Rng:        args.rng,
	}

	return mem
}

// Demand gets the demand stored in this Model.
// The fields returned can be filtered by passing resource.WithReadMask.
func (m *Model) Demand(opts ...resource.ReadOption) *traits.ElectricDemand {
	return m.demand.Get(opts...).(*traits.ElectricDemand)
}

// PullDemand subscribes to changes to the electricity demand on this device.
// The returned channel will be closed when done is called. You must call done after you are finished with the channel
// to prevents leaks and/or deadlocks. The channel will also be closed if ctx is cancelled.
func (m *Model) PullDemand(ctx context.Context, opts ...resource.ReadOption) <-chan PullDemandChange {
	send := make(chan PullDemandChange)

	recv := m.demand.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			demand := change.Value.(*traits.ElectricDemand)
			send <- PullDemandChange{
				Value:      demand,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send
}

// UpdateDemand will update the stored traits.ElectricDemand associated with this device.
// The fields to update can be filtered by passing resource.WithUpdateMask.
// The updated traits.ElectricDemand is returned.
func (m *Model) UpdateDemand(update *traits.ElectricDemand, opts ...resource.WriteOption) (*traits.ElectricDemand, error) {
	updated, err := m.demand.Set(update, opts...)
	if err != nil {
		return nil, err
	}
	return updated.(*traits.ElectricDemand), nil
}

// ActiveMode returns the electric mode that is currently active on this device.
// When the Model is first created, its active mode is a dummy mode with all-blank fields. After it is changed for
// the first time, it will always correspond to one of the modes that can be listed by Modes.
// The StartTime fields will reflect when the mode became active.
// The fields returned can be filtered using resource.WithReadMask
func (m *Model) ActiveMode(opts ...resource.ReadOption) *traits.ElectricMode {
	return m.activeMode.Get(opts...).(*traits.ElectricMode)
}

// PullActiveMode subscribes to changes to the active mode. Whenever the active mode is changed (for example, by calling
// ChangeActiveMode), the channel will send a notification.
// The returned channel will be closed when done is called. You must call done after you are finished with the channel
// to prevents leaks and/or deadlocks. The channel will also be closed if ctx is cancelled.
func (m *Model) PullActiveMode(ctx context.Context, opts ...resource.ReadOption) <-chan PullActiveModeChange {
	send := make(chan PullActiveModeChange)

	recv := m.activeMode.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			activeMode := change.Value.(*traits.ElectricMode)
			send <- PullActiveModeChange{
				ActiveMode: activeMode,
				ChangeTime: change.ChangeTime,
			}
		}
	}()

	// when done is called, then the resource will close recv for us
	return send
}

// SetActiveMode updates the active mode to the one specified.
// The mode.Id should exist in the known Modes of this model or an error will be returned.
// The mode.StartTime will not be set for you.
func (m *Model) SetActiveMode(mode *traits.ElectricMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.findMode(mode.Id); !ok {
		return ErrModeNotFound
	}

	_, err := m.activeMode.Set(mode)
	return err
}

// ChangeActiveMode will switch the active mode to a previously-defined mode with the given ID.
// Attempting to change to a mode ID that does not exist on this device will result in an error.
// Updates the StartTime of the mode to the current time if the mode changes.
func (m *Model) ChangeActiveMode(id string) (*traits.ElectricMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.changeActiveMode(id)
}

func (m *Model) changeActiveMode(id string) (*traits.ElectricMode, error) {
	mode, ok := m.findMode(id)
	if !ok {
		return nil, ErrModeNotFound
	}

	updated, err := m.activeMode.Set(mode, resource.InterceptAfter(func(old, new proto.Message) {
		oldMode := old.(*traits.ElectricMode)
		newMode := new.(*traits.ElectricMode)
		if oldMode.Id != newMode.Id {
			newMode.StartTime = timestamppb.New(m.clock.Now())
		}
	}))
	if err != nil {
		return nil, err
	}

	return updated.(*traits.ElectricMode), nil
}

// ChangeToNormalMode will (atomically) look up the device's normal mode (mode with Normal == true) and change to that
// mode.
// If this device does not have a normal mode, ErrModeNotFound is returned.
// Updates the StartTime of the mode to the current time if the mode changes.
func (m *Model) ChangeToNormalMode() (*traits.ElectricMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	normal, ok := m.normalMode()
	if !ok {
		return nil, ErrModeNotFound
	}

	return m.changeActiveMode(normal.Id)
}

// FindMode will attempt to retrieve the mode with the given ID.
// If the mode was found, it is returned with ok == true.
// Otherwise, the returned mode is unspecified and ok == false.
func (m *Model) FindMode(id string) (mode *traits.ElectricMode, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.findMode(id)
}

func (m *Model) findMode(id string) (*traits.ElectricMode, bool) {
	mode, ok := m.modes.Get(id)
	if !ok {
		return nil, false
	}
	return mode.(*traits.ElectricMode), true
}

// Modes returns a list of all registered modes, sorted by their ID.
func (m *Model) Modes(opts ...resource.ReadOption) []*traits.ElectricMode {
	entries := m.modes.List(opts...)

	modes := make([]*traits.ElectricMode, len(entries))
	for i, entry := range entries {
		modes[i] = entry.(*traits.ElectricMode)
	}
	return modes
}

// CreateMode adds a new mode to the device.
// The Id field on the mode must not be set, as the Id will be allocated by the device.
// If mode has Normal == true, and the device already has a normal mode, then ErrNormalModeExists will result.
// Returns the newly created mode, including its Id.
func (m *Model) CreateMode(mode *traits.ElectricMode) (*traits.ElectricMode, error) {
	if mode.Id != "" {
		// If the ID is set, this indicates a bug in the calling code
		panic("ID field is set")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.createOrAddMode(mode)
}

// AddMode adds a new mode to the device using the modes Id.
// The Id field on the mode must be set.
// If mode has Normal == true, and the device already has a normal mode, then ErrNormalModeExists will result.
func (m *Model) AddMode(mode *traits.ElectricMode) error {
	if mode.Id == "" {
		// If the ID is set, this indicates a bug in the calling code
		panic("ID field is not set")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.createOrAddMode(mode)
	return err
}

func (m *Model) createOrAddMode(mode *traits.ElectricMode) (*traits.ElectricMode, error) {
	// clone mode to avoid mutating the caller's copy
	mode = proto.Clone(mode).(*traits.ElectricMode)

	// if this mode is normal, check that there isn't another normal mode
	if mode.Normal {
		_, ok := m.normalMode()
		if ok {
			return nil, ErrNormalModeExists
		}
	}

	msg, err := m.modes.Add(mode.Id, mode, resource.WithGenIDIfAbsent(), resource.WithIDCallback(func(id string) {
		mode.Id = id
	}))
	if msg != nil {
		mode = msg.(*traits.ElectricMode)
	}
	return mode, err
}

// DeleteMode will remove the mode with the given Id from the device.
// If the mode does not exist, then ErrModeNotFound is returned.
// If the mode specified is the active mode, then ErrDeleteActiveMode is returned and the mode is not deleted.
// Otherwise, the operation succeeded and nil is returned.
func (m *Model) DeleteMode(id string, opts ...resource.WriteOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deleteMode(id, opts...)
}

func (m *Model) deleteMode(id string, opts ...resource.WriteOption) error {
	active := m.activeMode.Get().(*traits.ElectricMode)
	if id == active.Id {
		return ErrDeleteActiveMode
	}

	msg, err := m.modes.Delete(id, opts...)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrModeNotFound
	}

	return nil
}

// UpdateMode will modify one of the modes stored in this device.
// The mode to be modified is specified by mode.Id, which must be set.
// Fields to be modified can be selected using mask - to modify all fields, pass a nil mask.
func (m *Model) UpdateMode(mode *traits.ElectricMode, opts ...resource.WriteOption) (*traits.ElectricMode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateMode(mode, opts...)
}

func (m *Model) updateMode(mode *traits.ElectricMode, opts ...resource.WriteOption) (*traits.ElectricMode, error) {
	msg, err := m.modes.Update(mode.Id, mode, opts...)
	if err != nil {
		return nil, err
	}
	return msg.(*traits.ElectricMode), nil
}

// PullModes subscribes to changes to modes. Creation, modification or deletion of a mode on this device will send
// a PullModesChange describing the event down the changes channel.
// The returned channel will be closed when done is called. You must call done after you are finished with the channel
// to prevents leaks and/or deadlocks. The channel will also be closed if ctx is cancelled.
func (m *Model) PullModes(ctx context.Context, opts ...resource.ReadOption) <-chan PullModesChange {
	send := make(chan PullModesChange)
	recv := m.modes.Pull(ctx, opts...)

	go func() {
		defer close(send)
		// no need to listen to ctx.Done, as modes.Pull does that.
		for change := range recv {
			var newValue, oldValue *traits.ElectricMode
			if change.NewValue != nil {
				newValue = change.NewValue.(*traits.ElectricMode)
			}
			if change.OldValue != nil {
				oldValue = change.OldValue.(*traits.ElectricMode)
			}

			pullChange := PullModesChange{
				Type:       change.ChangeType,
				ChangeTime: change.ChangeTime,
				NewValue:   newValue,
				OldValue:   oldValue,
			}
			send <- pullChange
		}
	}()

	// when the caller invokes done, then recv will automatically be closed
	return send
}

// NormalMode returns the mode which has Normal == true. A device can have at most 1 such mode.
// If there is no normal mode on this device, then (nil, false) is returned.
func (m *Model) NormalMode() (*traits.ElectricMode, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.normalMode()
}

func (m *Model) normalMode() (*traits.ElectricMode, bool) {
	modes := m.modes.List()

	for _, mode := range modes {
		mode := mode.(*traits.ElectricMode)
		if mode.Normal {
			return mode, true
		}
	}

	return nil, false
}

type PullModesChange struct {
	Type       types.ChangeType
	NewValue   *traits.ElectricMode
	OldValue   *traits.ElectricMode
	ChangeTime time.Time
}

type PullDemandChange struct {
	Value      *traits.ElectricDemand
	ChangeTime time.Time
}

type PullActiveModeChange struct {
	ActiveMode *traits.ElectricMode
	ChangeTime time.Time
}
