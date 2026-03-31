package concurrent

import (
	"context"
	"maps"
	"sync"
	"time"
)

type Map[K comparable, V any] struct {
	m      sync.RWMutex
	values map[K]V
	bus    Bus[MapEvent[K, V]]
}

type MapEvent[K comparable, V any] struct {
	Type      MapEventType
	Key       K
	OldValue  V
	NewValue  V
	Timestamp time.Time
}

type MapEventType int

const (
	MapEventInsert MapEventType = iota + 1
	MapEventReplace
	MapEventDelete
)

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		values: make(map[K]V),
	}
}

// Get will retrieve the value corresponding to the given key. If the key is not present, returns the zero value
func (m *Map[K, V]) Get(key K) (value V, present bool) {
	m.m.RLock()
	defer m.m.RUnlock()

	value, present = m.values[key]
	return
}

// Copy creates a shallow copy of the map as a native Go map.
func (m *Map[K, V]) Copy() map[K]V {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.clone()
}

func (m *Map[K, V]) clone() map[K]V {
	cloned := make(map[K]V)
	maps.Copy(cloned, m.values)
	return cloned
}

// Store will store the value into the map under the given key. If there is an existing entry for the key, then
// the old value will be replaced and returned.
// If sending changes to all listeners succeeded before the context was cancelled, returns sent=true. Otherwise,
// sent=false and some listeners may not have received the notifications. The stored values, as retrieved by Get,
// will always be modified.
func (m *Map[K, V]) Store(ctx context.Context, key K, value V) (old V, present bool, sent bool) {
	m.m.Lock()
	old, present = m.values[key]
	m.values[key] = value
	timestamp := time.Now()
	m.m.Unlock()

	var ty MapEventType
	if present {
		ty = MapEventReplace
	} else {
		ty = MapEventInsert
	}

	sent = m.bus.Send(ctx, MapEvent[K, V]{
		Type:      ty,
		Key:       key,
		OldValue:  old,
		NewValue:  value,
		Timestamp: timestamp,
	})
	return
}

// GetOrStore will retrieve an entry in the map if it exists, otherwise creating it with the provided value.
func (m *Map[K, V]) GetOrStore(ctx context.Context, key K, store V) (value V, present bool, sent bool) {
	m.m.Lock()
	oldValue, present := m.values[key]
	timestamp := time.Now()
	if present {
		value = oldValue
	} else {
		m.values[key] = store
		value = store
	}
	m.m.Unlock()

	if !present {
		sent = m.bus.Send(ctx, MapEvent[K, V]{
			Type:      MapEventInsert,
			Key:       key,
			NewValue:  store,
			Timestamp: timestamp,
		})
	}
	return
}

func (m *Map[K, V]) Delete(ctx context.Context, key K) (old V, present bool, sent bool) {
	m.m.Lock()
	old, present = m.values[key]
	delete(m.values, key)
	timestamp := time.Now()
	m.m.Unlock()

	sent = m.bus.Send(ctx, MapEvent[K, V]{
		Type:      MapEventDelete,
		Key:       key,
		OldValue:  old,
		Timestamp: timestamp,
	})
	return
}

func (m *Map[K, V]) Events(ctx context.Context, backpressure bool, bufferSize int) <-chan MapEvent[K, V] {
	return m.bus.Listen(ctx, backpressure, bufferSize)
}

func (m *Map[K, V]) CloneAndEvents(ctx context.Context, backpressure bool, bufferSize int) (map[K]V, <-chan MapEvent[K, V]) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.clone(), m.Events(ctx, backpressure, bufferSize)
}
