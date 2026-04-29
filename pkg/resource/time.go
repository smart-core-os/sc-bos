package resource

import (
	"time"
)

// Clock defines all time related features required by this package.
type Clock interface {
	Now() time.Time
}

// WallClock returns a Clock backed by the time package.
func WallClock() Clock {
	return realClock{}
}

type realClock struct{}

func (r realClock) Now() time.Time {
	return time.Now()
}

type clockFunc func() time.Time

func (c clockFunc) Now() time.Time {
	return c()
}
