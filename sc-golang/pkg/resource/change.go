package resource

import (
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/masks"
)

// Comparer compares two messages for equivalence.
// This interface is used during the Pull operation to de-duplicate consecutive emissions.
type Comparer interface {
	// Compare compares two messages that changed at a specific time.
	// If ok is false then this Comparator did not attempt to compare the two messages, in other words the equal result
	// should be ignored.
	Compare(x, y proto.Message) bool
}

// ComparerFunc converts a func of the correct signature into a Comparer.
type ComparerFunc func(x, y proto.Message) bool

func (c ComparerFunc) Compare(x, y proto.Message) bool {
	return c(x, y)
}

// ValueChange contains information about a change to a Value.
type ValueChange struct {
	Value      proto.Message
	ChangeTime time.Time
	// Deprecated, use LastSeedValue instead.
	// SeedValue will be true if the change was part of sending initial data as opposed to an update.
	SeedValue bool
	// LastSeedValue will be true if this change is the last change as part of the seed values.
	LastSeedValue bool
}

func (v *ValueChange) filter(filter *masks.ResponseFilter) *ValueChange {
	newValue := filter.FilterClone(v.Value)
	if newValue == v.Value {
		return v
	}
	return &ValueChange{Value: newValue, ChangeTime: v.ChangeTime, SeedValue: v.SeedValue, LastSeedValue: v.LastSeedValue}
}

// CollectionChange contains information about a change to a Collection.
type CollectionChange struct {
	Id         string
	ChangeTime time.Time
	ChangeType types.ChangeType
	OldValue   proto.Message
	NewValue   proto.Message
	// Deprecated, use LastSeedValue instead.
	// SeedValue will be true if the change was part of sending initial data as opposed to an update.
	SeedValue bool
	// LastSeedValue will be true if this change is the last change as part of the seed values.
	LastSeedValue bool
}

func (c *CollectionChange) filter(filter *masks.ResponseFilter) *CollectionChange {
	newNewValue := filter.FilterClone(c.NewValue)
	newOldValue := filter.FilterClone(c.OldValue)
	if newNewValue == c.NewValue && newOldValue == c.OldValue {
		return c
	}
	return &CollectionChange{
		Id:            c.Id,
		ChangeType:    c.ChangeType,
		ChangeTime:    c.ChangeTime,
		OldValue:      newOldValue,
		NewValue:      newNewValue,
		SeedValue:     c.SeedValue,
		LastSeedValue: c.LastSeedValue,
	}
}

// include adjusts this change based on any filtering that is active on the collection.
// If items are being filtered, then an UPDATE that causes an items inclusion to change will report an ADD or REMOVE
// as needed. A new CollectionChange is returned if the underlying change type isn't accurate anymore.
// The ok return value will be true if an update should be sent, and false if the change shouldn't be forwarded on.
func (c *CollectionChange) include(includeFunc FilterFunc) (newChange *CollectionChange, ok bool) {
	if includeFunc == nil {
		return c, true
	}
	ret := func(c *CollectionChange, ok bool) (*CollectionChange, bool) {
		if !ok {
			return nil, false
		}
		return c, true
	}

	switch {
	case c.OldValue == nil && c.NewValue == nil:
		// this should never happen, but if it does just skip it
		return nil, false
	case c.OldValue == nil && c.NewValue != nil:
		// ADD
		return ret(c, includeFunc(c.Id, c.NewValue))
	case c.OldValue != nil && c.NewValue == nil:
		// REMOVE
		return ret(c, includeFunc(c.Id, c.OldValue))
	default:
		// update
	}

	oldInclude := includeFunc(c.Id, c.OldValue)
	newInclude := includeFunc(c.Id, c.NewValue)
	if oldInclude == newInclude {
		// the only time we want to skip sending the update is if both the old and new values are excluded
		return ret(c, newInclude)
	}

	if newInclude {
		// treat this like an Add
		return &CollectionChange{
			Id:         c.Id,
			ChangeType: types.ChangeType_ADD,
			ChangeTime: c.ChangeTime,
			NewValue:   c.NewValue,
			SeedValue:  c.SeedValue,
			// this is not safe, the caller needs to deal with this
			// LastSeedValue: c.LastSeedValue,
		}, true
	}

	// treat this like a remove
	return &CollectionChange{
		Id:         c.Id,
		ChangeType: types.ChangeType_REMOVE,
		ChangeTime: c.ChangeTime,
		OldValue:   c.OldValue,
	}, true
}
