package sim

import (
	"time"
)

// Updater publishes a slice of building state into a device's trait model.
// It is invoked on the engine goroutine after each Tick, so implementations may
// read Building state without locking.
type Updater interface {
	Update(now time.Time, b *Building)
}

// updaterFunc adapts a function to the Updater interface.
type updaterFunc func(now time.Time, b *Building)

func (f updaterFunc) Update(now time.Time, b *Building) { f(now, b) }

// ptr returns a pointer to v. Used for the many optional (pointer) proto scalar fields.
func ptr[T any](v T) *T { return &v }
