package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/pborman/uuid"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/maps"
)

var (
	ErrNotFound = errors.New("not found")
)

// Map tracks multiple Record with the ability to create, delete, and listen for changes to the tracked records.
type Map struct {
	mu      sync.Mutex
	known   map[string]*Record
	bus     *minibus.Bus[*Change]
	lastCID uint64 // tracks changes to known, guarded by mu. Events in bus record which cID they were created with.

	idFunc IdFunc
	create CreateFunc

	now func() time.Time
}

// CreateFunc returns a new Lifecycle for the given kind.
// CreateFunc is called during Map.Create.
type CreateFunc func(id, kind string) (Lifecycle, error)

// IdFunc generates a new id given the passed parameters.
// The function should return an error if no id can be found for which exists(id) returns false.
// IdFunc is called during Map.Create to mint new IDs where they are not provided as part of the call.
type IdFunc func(kind string, exists func(id string) bool) (string, error)

// NewMap creates and returns a new empty Map using the given create funcs.
func NewMap(createFunc CreateFunc, idFunc IdFunc) *Map {
	return &Map{
		known:  make(map[string]*Record),
		bus:    &minibus.Bus[*Change]{},
		idFunc: idFunc,
		create: createFunc,
		now:    time.Now,
	}
}

var ErrImmutable = errors.New("immutable")

// NewMapOf creates an immutable map containing only the given known services.
// Services will have an ID assigned based on the index in known.
// The returned map will return an error for Create and Delete.
func NewMapOf(known []Lifecycle) *Map {
	knownMap := make(map[string]*Record, len(known))
	for i, lifecycle := range known {
		id := strconv.FormatInt(int64(i), 10)
		kind := kindFromType(lifecycle)
		knownMap[id] = &Record{Id: id, Kind: kind, Service: lifecycle}
	}
	return &Map{
		known: knownMap,
		bus:   &minibus.Bus[*Change]{},
		now:   time.Now,
	}
}

func kindFromType(t any) string {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt.String()
}

// MapSetNow sets the now func on m returning a func that undoes the set.
// Use for testing.
func MapSetNow(m *Map, now func() time.Time) func() {
	old := m.now
	m.now = now
	return func() {
		m.now = old
	}
}

var (
	// IdIsKind is an IdFunc that attempts to use the kind as the id.
	// This implies that only one service of a given type can exist in the map.
	IdIsKind = IdFunc(func(kind string, exists func(id string) bool) (string, error) {
		if exists(kind) {
			return "", errors.New("id exists")
		}
		return kind, nil
	})
	IdIsUUID = IdFunc(func(kind string, exists func(id string) bool) (string, error) {
		max := 10
		for range max {
			id := uuid.New()
			if exists(id) {
				continue
			}
			return id, nil
		}
		return "", errors.New("exhausted attempts finding unique uuid")
	})
	IdIsRequired = IdFunc(func(_ string, _ func(id string) bool) (string, error) {
		return "", errors.New("id is required")
	})
)

type Record struct {
	Id, Kind string
	Service  Lifecycle
}

// Create creates and adds a new record to m returning the new ID and the records service State.
// The kind argument is required, but id is optional. If absent the IdFunc will be used to mint a new ID.
// Only State.Active and State.Config are optionally used in the passed state to either Start or Configure the created
// Lifecycle. If either are present and the corresponding Lifecycle call returns an error, then creating the new record
// will be aborted and that error will be returned.
func (m *Map) Create(id, kind string, state State) (string, State, error) {
	r, cID, err := m.createRecord(id, kind)
	if err != nil {
		return "", State{}, err
	}

	outState := r.Service.State()
	defer func() {
		if err != nil {
			m.mu.Lock()
			defer m.mu.Unlock()

			// cleanup the known map
			got, ok := m.known[r.Id]
			if ok && got == r {
				delete(m.known, r.Id)
			}

			// cleanup the service
			if outState.Active {
				_, _ = r.Service.Stop()
			}
		}
	}()

	if state.Active {
		outState, err = r.Service.Start()
	}
	if len(state.Config) > 0 {
		outState, err = r.Service.Configure(state.Config)
	}

	change := &Change{
		ChangeTime: m.now(),
		ChangeType: types.ChangeType_ADD,
		NewValue:   r,
		cID:        cID,
	}
	go m.bus.Send(context.Background(), change)

	return r.Id, outState, err
}

// Delete stops and removes the record with the given ID from m.
// If id is not found, returns ErrNotFound.
// Delete does not error if the record is already stopped.
func (m *Map) Delete(id string) (State, error) {
	r, cID, err := m.deleteRecord(id)
	if err != nil {
		return State{}, err
	}

	// stop before sending the event
	state, err := r.Service.Stop()
	if errors.Is(err, ErrAlreadyStopped) {
		err = nil
	}

	go m.bus.Send(context.Background(), &Change{
		ChangeTime: m.now(),
		ChangeType: types.ChangeType_REMOVE,
		OldValue:   r,
		cID:        cID,
	})
	return state, err
}

// Get returns the record associated with the given id, or nil if not found.
func (m *Map) Get(id string) *Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.known[id]
}

// Values returns all known values in an indeterminate order.
func (m *Map) Values() []*Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Values(m.known)
}

// States returns all known states in an indeterminate order.
func (m *Map) States() []*StateRecord {
	m.mu.Lock()
	records := maps.Values(m.known)
	m.mu.Unlock()
	out := make([]*StateRecord, len(records))
	for i, val := range records {
		r := &StateRecord{
			Record: val,
			State:  val.Service.State(),
		}
		out[i] = r
	}
	return out
}

// Listen emits changes to m on the returns chan until ctx is done.
func (m *Map) Listen(ctx context.Context) <-chan *Change {
	return m.bus.Listen(ctx)
}

func (m *Map) GetAndListen(ctx context.Context) ([]*Record, <-chan *Change) {
	// must listen before getting values
	ch := m.bus.Listen(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	values := maps.Values(m.known)
	cID := m.lastCID // the id of the event that is associated with values

	// skip events that are already recorded in values.
	// this avoids, for example, having an item in values that also has a CREATED event waiting in bus.
	// The main reason for this is to avoid holding mu while sending on bus as that's blocking based on caller code
	out := make(chan *Change)
	go func() {
		defer close(out)
		for change := range ch {
			if change.cID == cID+1 {
				cID++
				if err := chans.SendContext(ctx, out, change); err != nil {
					return
				}
			}
		}
	}()
	return values, out
}

func (m *Map) GetAndListenState(ctx context.Context) ([]*StateRecord, <-chan *StateChange) {
	var states []*StateRecord
	out := make(chan *StateChange)
	var wg sync.WaitGroup // tracks go routines that can send to out, we only close out when there are no more
	listen := func(record *Record, stateRecord *StateRecord, changes <-chan State) {
		defer wg.Done()
		m.listenRecordStates(ctx, record, stateRecord, changes, out)
	}

	records, changes := m.GetAndListen(ctx)
	stopByID := make(map[string]context.CancelFunc)

	// current values
	for _, record := range records {
		ctx, stop := context.WithCancel(ctx)
		stopByID[record.Id] = stop

		state, stateChanges := record.Service.StateAndChanges(ctx)
		stateRecord := &StateRecord{
			Record: record,
			State:  state,
		}
		states = append(states, stateRecord)
		wg.Add(1)
		go listen(record, stateRecord, stateChanges)
	}

	// updates
	go func() {
		for change := range changes {
			switch {
			case change.OldValue == nil && change.NewValue == nil: // just in case
			case change.OldValue == nil: // add
				// just in case
				if stop, ok := stopByID[change.NewValue.Id]; ok {
					log.Printf("state listener for %s already exists, stopping", change.NewValue.Id)
					stop()
				}

				// this ctx tracks the listener on the records state
				ctx, stop := context.WithCancel(ctx)
				stopByID[change.NewValue.Id] = stop

				record := change.NewValue
				state, stateChanges := record.Service.StateAndChanges(ctx)
				stateRecord := &StateRecord{
					Record: record,
					State:  state,
				}
				stateChange := &StateChange{
					OldValue:   nil,
					NewValue:   stateRecord,
					ChangeTime: change.ChangeTime,
					ChangeType: types.ChangeType_ADD,
				}
				if err := chans.SendContext(ctx, out, stateChange); err != nil {
					return
				}
				wg.Add(1)
				go listen(record, stateRecord, stateChanges)
			case change.NewValue == nil: // remove
				if stop, ok := stopByID[change.OldValue.Id]; ok {
					stop()
					delete(stopByID, change.OldValue.Id)
				}
			}
		}

		// ctx is done, that's the only way changes is closed
		wg.Wait()
		close(out)
	}()

	return states, out
}

// listenRecordStates sends on out each time stateChanges sends.
// ctx should be the context for the overarching listen, the one associated with _all_ records.
// stateChanges should close when no more changes are going to be sent.
// This function will send a REMOVE to out before it returns, so out should not be closed until then.
func (m *Map) listenRecordStates(ctx context.Context, record *Record, stateRecord *StateRecord, stateChanges <-chan State, out chan<- *StateChange) {
	for newState := range stateChanges {
		old := stateRecord
		stateRecord = &StateRecord{
			Record: record,
			State:  newState,
		}
		change := &StateChange{
			OldValue:   old,
			NewValue:   stateRecord,
			ChangeType: types.ChangeType_UPDATE,
			ChangeTime: m.now(),
		}
		if err := chans.SendContext(ctx, out, change); err != nil {
			return
		}
	}

	removedChange := &StateChange{
		OldValue:   stateRecord,
		ChangeType: types.ChangeType_REMOVE,
		ChangeTime: m.now(),
	}
	// ignore the error because this is the last thing we're doing anyway
	_ = chans.SendContext(ctx, out, removedChange)
}

func (m *Map) createRecord(id, kind string) (*Record, uint64, error) {
	if m.create == nil {
		return nil, 0, ErrImmutable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// gen id if needed
	if id == "" {
		var err error
		id, err = m.idFunc(kind, m.idExists)
		if err != nil {
			return nil, 0, fmt.Errorf("bad id %w", err)
		}
	}

	s, err := m.create(id, kind)
	if err != nil {
		return nil, 0, err
	}

	r := &Record{
		Id:      id,
		Kind:    kind,
		Service: s,
	}
	m.known[id] = r
	m.lastCID++
	return r, m.lastCID, nil
}

func (m *Map) deleteRecord(id string) (*Record, uint64, error) {
	if m.create == nil {
		return nil, 0, ErrImmutable
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.known[id]
	if !ok {
		return nil, 0, ErrNotFound
	}
	delete(m.known, id)
	m.lastCID++
	return r, m.lastCID, nil
}

// idExists returns whether the given id exists in m.known.
// m.mu lock must be held when calling this method
func (m *Map) idExists(id string) bool {
	return maps.Exists(m.known, id)
}

// Change represents a change to a Map.
type Change struct {
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   *Record
	NewValue   *Record
	cID        uint64
}

type StateRecord struct {
	*Record
	State State
}

type StateChange struct {
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   *StateRecord
	NewValue   *StateRecord
}
