package time

import (
	"github.com/smart-core-os/sc-api/go/types/time"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// cut is an internal interface we use for dividing up a timeline. It has 4 variants belowAll, below, above, and
// aboveAll. With these options we can define any range on the timeline.
//
// This pattern is borrowed from Guavas Range
type cut interface {
	CompareTo(that cut) int
}

// cutPeriod 'cuts' the timeline based on the given Period. nil values in the Period represent unbounded/absent cuts
func cutPeriod(p *time.Period) (lower cut, upper cut) {
	if p.StartTime == nil && p.EndTime == nil {
		return cutBelowAll(), cutAboveAll()
	} else if p.StartTime == nil {
		return cutBelowAll(), cutBelow(p.EndTime)
	} else if p.EndTime == nil {
		return cutBelow(p.StartTime), cutAboveAll()
	} else {
		return cutBelow(p.StartTime), cutBelow(p.EndTime)
	}
}

func cutBelow(ts *timestamppb.Timestamp) cut {
	return (*below)(ts)
}

func cutAbove(ts *timestamppb.Timestamp) cut {
	return (*above)(ts)
}

func cutAboveAll() cut {
	return aboveAllInstance
}

func cutBelowAll() cut {
	return belowAllInstance
}

func compareValueCuts(this, that cut) int {
	if _, ok := that.(*belowAll); ok {
		return 1
	}
	if _, ok := that.(*aboveAll); ok {
		return -1
	}

	thisTs, thisIsAbove := extractValue(this)
	thatTs, thatIsAbove := extractValue(that)
	result := CompareAscending(thisTs, thatTs)
	if result != 0 {
		return result
	}

	// same value, below comes before above
	if thisIsAbove == thatIsAbove {
		return 0
	} else if thisIsAbove {
		return 1
	} else {
		return -1
	}
}

func extractValue(c cut) (ts *timestamppb.Timestamp, isAbove bool) {
	if thatBelow, ok := c.(*below); ok {
		return (*timestamppb.Timestamp)(thatBelow), false
	}
	if thatAfter, ok := c.(*above); ok {
		return (*timestamppb.Timestamp)(thatAfter), true
	}
	panic("unexpected type")
}

// below represents a cut on the timeline below the defined timestamp
type below timestamppb.Timestamp

func (b *below) CompareTo(that cut) int {
	return compareValueCuts(b, that)
}

// above represents a cut on the timeline above the defined timestamp
type above timestamppb.Timestamp

func (b *above) CompareTo(that cut) int {
	return compareValueCuts(b, that)
}

// belowAll represents a cut on the timeline below any possible timestamp
type belowAll struct{}

var belowAllInstance = &belowAll{}

func (c *belowAll) CompareTo(that cut) int {
	if that == belowAllInstance {
		return 0
	}
	return -1
}

// aboveAll represents a cut on the timeline above any possible timestamp
type aboveAll struct{}

var aboveAllInstance = &aboveAll{}

func (c *aboveAll) CompareTo(that cut) int {
	if that == aboveAllInstance {
		return 0
	}
	return 1
}
