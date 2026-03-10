package enterleavesensorpb

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Model provides the data structure for representing an EnterLeaveSensor trait.
type Model struct {
	enterLeaveEvents *resource.Value // of *traits.EnterLeaveEvent
}

// NewModel constructs a new model with no prior enter or leave information.
func NewModel(opts ...resource.Option) *Model {
	args := calcModelArgs(opts...)
	return &Model{
		enterLeaveEvents: resource.NewValue(args.enterLeaveEventsOpts...),
	}
}

// CreateEnterLeaveEvent creates and publishes a new EnterLeaveEvent to Pull subscribers.
// The last call to CreateEnterLeaveEvent is the one that is used as the current state if Pull is called with a false
// updates_only field.
// EnterTotal and LeaveTotal will be adjusted automatically if
//  1. No InterceptorBefore option is provided
//  2. The events direction is either ENTER or LEAVE
//  3. The event has EnterTotal or LeaveTotal equal to the current value or nil. If non-nil then the totals will be set to the passed values.
func (m *Model) CreateEnterLeaveEvent(event *traits.EnterLeaveEvent, opts ...resource.WriteOption) error {
	updateTotalsOption := resource.InterceptBefore(func(current, value proto.Message) {
		currentVal, valueVal := current.(*traits.EnterLeaveEvent), value.(*traits.EnterLeaveEvent)
		adjustTotal := func(val, cur *int32, inc bool) *int32 {
			var cv int32
			if cur != nil {
				cv = *cur
			}
			if val != nil && *val != cv {
				// the caller supplied a new total, use it
				return val
			}
			if inc {
				cv++
			}
			return &cv
		}
		valueVal.EnterTotal = adjustTotal(valueVal.EnterTotal, currentVal.EnterTotal, valueVal.Direction == traits.EnterLeaveEvent_ENTER)
		valueVal.LeaveTotal = adjustTotal(valueVal.LeaveTotal, currentVal.LeaveTotal, valueVal.Direction == traits.EnterLeaveEvent_LEAVE)
	})
	opts = append([]resource.WriteOption{updateTotalsOption}, opts...)
	_, err := m.enterLeaveEvents.Set(event, opts...)
	return err
}

type EnterLeaveEventChange struct {
	ChangeTime time.Time
	Value      *traits.EnterLeaveEvent
}

// PullEnterLeaveEvents subscribes to changes in the enter leave sensor resource.
// The returned chan will be closed when the given context is Done.
func (m *Model) PullEnterLeaveEvents(ctx context.Context, opts ...resource.ReadOption) <-chan EnterLeaveEventChange {
	send := make(chan EnterLeaveEventChange)
	go func() {
		defer close(send)
		for change := range m.enterLeaveEvents.Pull(ctx, opts...) {
			val := change.Value.(*traits.EnterLeaveEvent)
			if change.LastSeedValue {
				// when sending the initial data (not an update), the occupant and direction should be absent.
				val.Occupant = nil
				val.Direction = traits.EnterLeaveEvent_DIRECTION_UNSPECIFIED
			}
			select {
			case <-ctx.Done():
				return
			case send <- EnterLeaveEventChange{ChangeTime: change.ChangeTime, Value: val}:
			}
		}
	}()
	return send
}

func (m *Model) GetEnterLeaveEvent(opts ...resource.ReadOption) (*traits.EnterLeaveEvent, error) {
	val := m.enterLeaveEvents.Get(opts...)
	return val.(*traits.EnterLeaveEvent), nil
}

func (m *Model) ResetTotals() error {
	var zero int32
	_, err := m.enterLeaveEvents.Set(&traits.EnterLeaveEvent{EnterTotal: &zero, LeaveTotal: &zero}, resource.WithUpdatePaths("enter_total", "leave_total"))
	return err
}
