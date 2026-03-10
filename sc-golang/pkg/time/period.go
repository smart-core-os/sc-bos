package time

import (
	"github.com/smart-core-os/sc-api/go/types/time"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PeriodsConnected returns true if there exists a (possibly empty) Period that is enclosed by both p1 and p2
//
// For example
//  * `[2, 4)` and `[5, 7)` are not connected
//  * `[2, 4)` and `[3, 5)` are connected, because both enclose `[3, 4)`
//  * `[2, 4)` and `[4, 6)` are connected, because both enclose the empty period `[4, 4)`
func PeriodsConnected(p1, p2 *time.Period) bool {
	if p1 == nil || p2 == nil {
		return false
	}

	p1lower, p1upper := cutPeriod(p1)
	p2lower, p2upper := cutPeriod(p2)

	return p1lower.CompareTo(p2upper) <= 0 &&
		p2lower.CompareTo(p1upper) <= 0
}

// PeriodsIntersect returns true if there exists a non-empty Period that is enclosed by both p1 and p2
//
// For example
//  * `[2, 4)` and `[5, 7)` do not intersect
//  * `[2, 4)` and `[3, 5)` intersect, because both enclose `[3, 4)` which is non-empty
//  * `[2, 4)` and `[4, 6)` do not intersect, because both enclose the empty period `[4, 4)`
func PeriodsIntersect(p1, p2 *time.Period) bool {
	if p1 == nil || p2 == nil {
		return false
	}

	p1lower, p1upper := cutPeriod(p1)
	p2lower, p2upper := cutPeriod(p2)

	return p1lower.CompareTo(p2upper) < 0 &&
		p2lower.CompareTo(p1upper) < 0
}

func AllTime() *time.Period {
	return &time.Period{}
}

func PeriodBetween(t1, t2 *timestamppb.Timestamp) *time.Period {
	return &time.Period{StartTime: t1, EndTime: t2}
}

func PeriodBefore(t *timestamppb.Timestamp) *time.Period {
	return &time.Period{EndTime: t}
}

func PeriodOnOrAfter(t *timestamppb.Timestamp) *time.Period {
	return &time.Period{StartTime: t}
}
