package segmentpb

import (
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

// ActiveAt finds the active segment at time d.
// The index of the active segment in segments is returned along with the elapsed time before that segment started.
// If segments is empty, (0,0) is returned. If d<0, (d,0) is returned.
// If d is after all segments, the total length of the segments is returned as elapsed, and len(segments) as index.
func ActiveAt(d time.Duration, segments ...*traits.ElectricMode_Segment) (elapsed time.Duration, index int) {
	if d < 0 {
		return d, 0
	}

	var cur time.Duration
	for i, segment := range segments {
		if segment.Length == nil {
			return cur, i
		}

		l := segment.Length.AsDuration()
		if cur+l > d {
			return cur, i
		}
		cur += l
	}

	return cur, len(segments)
}
