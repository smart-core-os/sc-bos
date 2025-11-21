package hikcentral

import (
	"container/ring"
	"sync"
)

type Ring[T any] struct {
	// oldest value in the ring is at head
	head *ring.Ring
	// newest value is at tail
	tail *ring.Ring
	// protects head and tail
	mtx sync.Mutex
}

func NewRing[T any](size int) *Ring[T] {
	if size <= 0 {
		size = 1
	}
	r := ring.New(size)
	return &Ring[T]{
		head: r,
		tail: r,
	}
}

func (r *Ring[T]) Add(value T) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.tail.Value = value
	r.tail = r.tail.Next()
	if r.tail == r.head {
		r.head = r.head.Next()
	}
}

func (r *Ring[T]) Values() []T {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	var values []T
	r.head.Do(func(a any) {
		if a == nil {
			return
		}
		values = append(values, a.(T))
	})

	return values
}

func (r *Ring[T]) Len() int {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	count := 0
	r.head.Do(func(a any) {
		if a != nil {
			count++
		}
	})
	return count
}

func (r *Ring[T]) Find(predicate func(T) bool) T {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	var result T
	r.head.Do(func(a any) {
		if a == nil {
			return
		}
		v := a.(T)
		if predicate(v) {
			result = v
			return
		}
	})

	return result
}

func (r *Ring[T]) Update(predicate func(T) bool, updater func(T)) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.head.Do(func(a any) {
		if a == nil {
			return
		}
		v := a.(T)
		if predicate(v) {
			updater(v)
			return
		}
	})
}
