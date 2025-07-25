package minirx

import (
	"context"
	"iter"
	"maps"
	"sync"
)

type Map[K comparable, V any] struct {
	m         sync.RWMutex
	data      map[K]V
	listeners map[*mapListener[K, V]]struct{}
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		data:      make(map[K]V),
		listeners: make(map[*mapListener[K, V]]struct{}),
	}
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	m.m.RLock()
	defer m.m.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func (m *Map[K, V]) Put(key K, value V) {
	m.m.Lock()
	defer m.m.Unlock()

	changeType := Add
	if _, exists := m.data[key]; exists {
		changeType = Update
	}

	m.data[key] = value
	m.dispatchChange(key, value, changeType)
}

func (m *Map[K, V]) Delete(key K) {
	m.m.Lock()
	defer m.m.Unlock()

	if _, exists := m.data[key]; !exists {
		return
	}

	delete(m.data, key)
	var empty V
	m.dispatchChange(key, empty, Delete)
}

func (m *Map[K, V]) dispatchChange(key K, value V, changeType MapChange) {
	for listener := range m.listeners {
		listener.notify(key, changeType, value)
	}
}

func (m *Map[K, V]) copyData() map[K]V {
	result := make(map[K]V, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

func (m *Map[K, V]) Pull(ctx context.Context) (initial map[K]V, changes iter.Seq[MapChanges[K, V]]) {
	m.m.Lock()
	defer m.m.Unlock()

	initial = m.copyData()
	l := &mapListener[K, V]{}
	l.c = sync.Cond{L: &l.m}
	l.cleanupAfterFunc = context.AfterFunc(ctx, l.Stop)
	m.listeners[l] = struct{}{}

	return initial, l.Iter
}

type MapChanges[K comparable, V any] struct {
	Current map[K]V
	Changed map[K]MapChange
}

func (m *MapChanges[K, V]) merge(k K, v V, change MapChange) {
	if change == Delete {
		delete(m.Current, k)
	} else {
		m.Current[k] = v
	}

	// correct change type
	// may be necessary if we're stacking multiple changes together for the same key
	existingChange, ok := m.Changed[k]
	if ok {
		_ = existingChange
		panic("unimplemented")
	} else {
		m.Changed[k] = change
	}
}

type MapChange int

const (
	Add MapChange = iota + 1
	Delete
	Update
)

type mapListener[K comparable, V any] struct {
	cleanupAfterFunc func() bool

	m       sync.Mutex
	c       sync.Cond
	stopped bool
	current map[K]V
	changes map[K]MapChange
}

func (l *mapListener[K, V]) Iter(yield func(MapChanges[K, V]) bool) {
	l.m.Lock()
	defer l.m.Unlock()
	for {
		if l.stopped {
			return
		}
		if len(l.changes) > 0 {
			mapChanges := MapChanges[K, V]{
				Current: maps.Clone(l.current),
				Changed: l.changes,
			}
			l.changes = make(map[K]MapChange) // reset changes for next iteration

			l.m.Unlock()
			keepIterating := yield(mapChanges)
			l.m.Lock()

			if !keepIterating {
				l.stopped = true
				return
			}
		} else {
			l.c.Wait()
		}
	}
}

func (l *mapListener[K, V]) Stop() {
	_ = l.cleanupAfterFunc()
	l.m.Lock()
	l.stopped = true
	l.m.Unlock()
	l.c.Broadcast()
}

func (l *mapListener[K, V]) notify(k K, change MapChange, newV V) (open bool) {
	l.m.Lock()
	defer l.m.Unlock()

	if l.stopped {
		return false
	}

	l.c.Broadcast()
	panic("unimplemented") // TODO: implement this
}
