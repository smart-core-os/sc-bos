package segmentpb

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-api/go/traits"
)

// MaxMagnitude returns the largest segment magnitude of all non-zero length segments.
// If segments is empty or contains only zero-length segments, returns 0.
func MaxMagnitude(segments ...*traits.ElectricMode_Segment) (max float32) {
	i := Max(segments...)
	if i < len(segments) {
		return segments[i].Magnitude
	}
	return 0
}

// Max returns the index of the segment with the largest magnitude of all non-zero length segments.
// If segments is empty, or contains only zero length segments, len(segments) is returned.
func Max(segments ...*traits.ElectricMode_Segment) (index int) {
	var found bool
	var max float32
	for i, segment := range segments {
		if segment.Length != nil && !durationPositive(segment.Length) {
			continue // don't count zero length segments
		}
		if !found {
			max = segment.Magnitude
			index = i
			found = true
		} else if segment.Magnitude > max {
			max = segment.Magnitude
			index = i
		}
	}
	if !found {
		return len(segments)
	}
	return index
}

// MaxAfter returns the index of the segment with the largest magnitude of all non-zero length segments after d time.
// If segments is empty, or contains only zero length segments, len(segments) is returned.
func MaxAfter(d time.Duration, segments ...*traits.ElectricMode_Segment) (index int) {
	_, i := ActiveAt(d, segments...)
	return Max(segments[i:]...) + i
}

// SumMagnitude sums the magnitude of all the given segments.
func SumMagnitude(segments ...*traits.ElectricMode_Segment) (sum float32) {
	for _, segment := range segments {
		sum += segment.Magnitude
	}
	return sum
}

// MagnitudeAt returns the magnitude of the segment active at d.
// If there is no segment at d, ok will be false.
func MagnitudeAt(d time.Duration, segments ...*traits.ElectricMode_Segment) (level float32, ok bool) {
	if d < 0 {
		return 0, false
	}
	_, i := ActiveAt(d, segments...)
	if i >= len(segments) {
		return 0, false
	}
	return segments[i].Magnitude, true
}

func durationPositive(d *durationpb.Duration) bool {
	return d.AsDuration() > 0
}
