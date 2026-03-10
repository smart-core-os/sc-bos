package segmentpb

import (
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

// Duration returns the total duration for all the given segments.
// If a segment is found without a length, the total up to that point and true will be returned.
func Duration(s ...*traits.ElectricMode_Segment) (total time.Duration, infinite bool) {
	for _, segment := range s {
		l := segment.GetLength()
		if l == nil {
			return total, true // the last segment is the only one allowed to have a nil length
		}
		total += l.AsDuration()
	}
	return total, false
}
