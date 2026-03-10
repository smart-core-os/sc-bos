package clock

import (
	"time"
)

// Clock represents an abstracted time interface. The Real function returns the trivial implementation based on
// the time package, that operates on real, wall-clock time. Other implementations may allow time to run at different
// speeds, or completely decoupled from real time, for simulation purposes.
//
// A Clock may be stopped in an implementation-defined manner. Once the Clock stops, it cannot be used further.
// When this occurs, all open channels returned from this interface will be closed. API consumers should
// check for closes, if they need to distinguish between a real event and the Clock stopping.
//
// A Clock is not guaranteed to send on channels at exactly the times requested. For example, if it operates
// on fixed timesteps, the times may be quantised. Functions/channels report what time the event actually occurred at.
type Clock interface {
	// Now returns the current instant, in Clock time.
	Now() time.Time

	// At returns a channel that sends a single value as soon as possible at or after time t. If the Clock is stopped
	// before the channel sends, the channel will be closed. If t is before Now, the channel will send as soon as possible.
	// The channel value is the actual time that the clock tried to send at. This could be different to t,
	// if the Clock has limited precision.
	// The channel will have a buffer size of 1, and will be sent through at most once, so discarding it will not cause
	// leaks or deadlocks.
	At(t time.Time) <-chan time.Time
	// After returns a channel that sends a single value as soon as possible at or after the duration d has elapsed,
	// relative to Now. If the Clock is stopped before the channel sends, the channel will be closed.
	// The channel value is the actual time the clock sent. It could be different to Now().Add(d) if the Clock
	// has limited precision.
	// The channel will have a buffer size of 1, and will be sent through at most once, so discarding it will
	// not cause leaks or deadlocks.
	// If d is negative, sends through the channel as soon as possible.
	After(d time.Duration) <-chan time.Time
	// Every returns a channel c that sends a value every time the duration d elapses. When the Clock is stopped before
	// the ticker is stopped, then the output channel Ticker.C() is closed.
	// Panics if d is negative.
	// You must stop the Ticker after you are done using it by calling Ticker.Stop. Implementations are permitted
	// to deadlock or leak if you do not do so.
	Every(d time.Duration) Ticker
}

type Ticker interface {
	// C retrieves the channel that can be used to receive the ticks.
	C() <-chan time.Time
	// Stop is used to stop any current or future sends on C. You must call Stop exactly once on every Ticker.
	// Implementations are not required to permit multiple calls to Stop.
	Stop()
}

// Real returns a clock based on wall-clock time, using the standard time package.
// This clock cannot be stopped.
// All methods are thread-safe.
func Real() Clock {
	return &realClock{}
}

type realClock struct{}

func (c *realClock) Now() time.Time {
	return time.Now()
}

func (c *realClock) At(t time.Time) <-chan time.Time {
	return c.After(t.Sub(c.Now()))
}

func (c *realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c *realClock) Every(d time.Duration) Ticker {
	return realTicker{time.NewTicker(d)}
}

type realTicker struct {
	*time.Ticker
}

func (t realTicker) C() <-chan time.Time {
	return t.Ticker.C
}
