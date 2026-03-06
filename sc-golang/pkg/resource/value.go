package resource

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-golang/internal/minibus"
)

// Value represents a simple state field in an object. Think Temperature or Volume or Occupancy. Use a Value to
// gain thread safe reads/writes that also support FieldMasks and update notifications.
type Value struct {
	*config

	mu         sync.RWMutex
	value      proto.Message
	changeTime time.Time

	bus minibus.Bus
}

func NewValue(opts ...Option) *Value {
	c := computeConfig(opts...)
	res := &Value{
		config: c,
	}
	res.value = c.initialValue
	res.changeTime = c.clock.Now()
	c.initialValue = nil // clear so it can be GC'd when the value changes
	return res
}

func (r *Value) Get(opts ...ReadOption) proto.Message {
	return r.get(ComputeReadConfig(opts...))
}

func (r *Value) get(req *ReadRequest) proto.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return req.FilterClone(r.value)
}

// Set updates the current value of this Value with the given value.
// Returns the new value.
// Provide WriteOption to control masks and other variables during the update.
func (r *Value) Set(value proto.Message, opts ...WriteOption) (proto.Message, error) {
	return r.set(value, ComputeWriteConfig(opts...))
}

func (r *Value) set(value proto.Message, request WriteRequest) (proto.Message, error) {
	writer := request.fieldUpdater(r.writableFields)
	if err := writer.Validate(value); err != nil {
		return nil, err
	}

	disarm := timeoutAlarm(time.Second, "GetAndUpdate took too long")
	_, newValue, err := GetAndUpdate(
		&r.mu,
		func() (proto.Message, error) {
			return r.value, nil
		},
		request.changeFn(writer, value),
		func(message proto.Message) {
			r.value = message
			r.changeTime = request.updateTime(r.clock)
		},
	)
	disarm()

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()
	r.bus.Send(ctx, &ValueChange{
		Value:      newValue,
		ChangeTime: request.updateTime(r.clock),
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, errors.New("bus.Send blocked for too long")
	}

	return newValue, err
}

// Pull emits a ValueChange on the returned chan whenever the underlying value changes.
// The changes emitted can be adjusted using WithEquivalence.
// The returned chan will be closed when no more events will be emitted, either because ctx was cancelled or for other
// reasons.
func (r *Value) Pull(ctx context.Context, opts ...ReadOption) <-chan *ValueChange {
	readConfig := ComputeReadConfig(opts...)
	filter := readConfig.ResponseFilter()
	on, currentValue, changeTime := r.onUpdate(ctx, readConfig)
	typedEvents := make(chan *ValueChange)
	go func() {
		defer close(typedEvents)

		if currentValue != nil {
			change := &ValueChange{Value: currentValue, ChangeTime: changeTime, SeedValue: true, LastSeedValue: true}
			change = change.filter(filter)
			select {
			case <-ctx.Done():
				return // give up sending
			case typedEvents <- change:
			}
		}

		last := currentValue
		for event := range on {
			change := event.(*ValueChange).filter(filter)
			if r.equivalence != nil && r.equivalence.Compare(last, change.Value) {
				continue
			}
			last = change.Value
			select {
			case <-ctx.Done():
				return // give up sending
			case typedEvents <- change:
			}
		}
	}()
	return typedEvents
}

func (r *Value) onUpdate(ctx context.Context, config *ReadRequest) (<-chan any, proto.Message, time.Time) {
	var (
		value      proto.Message
		changeTime time.Time
	)
	if !config.UpdatesOnly {
		r.mu.RLock()
		defer r.mu.RUnlock()
		value = r.value
		changeTime = r.changeTime
	}

	ch := r.bus.Listen(ctx)
	if !config.Backpressure {
		ch = minibus.DropExcess(ch)
	}

	return ch, value, changeTime
}

func timeoutAlarm(duration time.Duration, fmt string, args ...any) (disarm func()) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)

	go func() {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Printf(fmt, args...)
		}
	}()

	return cancel
}

// Clock returns the clock used by this resource for reporting time.
func (r *Value) Clock() Clock {
	return r.clock
}
