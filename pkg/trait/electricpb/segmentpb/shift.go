package segmentpb

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-api/go/traits"
)

// Shift adjusts the offset of the given segments by d.
// If d is positive, this is the equivalent of inserting a new 0-magnitude segment of length d to the start.
// If d is negative, this removes segments until d length has been removed. This will cut the first returned segment if
// d falls within the length of that segment.
func Shift(d time.Duration, segments ...*traits.ElectricMode_Segment) []*traits.ElectricMode_Segment {
	if d == 0 || len(segments) == 0 {
		return segments // simple case
	}

	if d > 0 {
		// attempt to extend the magnitude of the first slice instead of prepending a new one
		if first := segments[0]; first.Magnitude == 0 {
			if first.Length == nil {
				return segments // if the first value is infinite, adding to it makes no sense
			}
			first = proto.Clone(first).(*traits.ElectricMode_Segment) // clone so we don't update the original
			first.Length = durationpb.New(first.Length.AsDuration() + d)
			out := make([]*traits.ElectricMode_Segment, len(segments))
			out[0] = first
			copy(out[1:], segments[1:])
			return out
		}

		out := make([]*traits.ElectricMode_Segment, len(segments)+1)
		out[0] = &traits.ElectricMode_Segment{Length: durationpb.New(d)}
		copy(out[1:], segments)
		return out
	}

	var cur time.Duration
	d = -d
	for i, segment := range segments {
		if segment.Length == nil {
			return segments[i:]
		}
		l := segment.Length.AsDuration()
		if cur+l > d {
			_, after, _ := Cut(d-cur, segment)
			out := make([]*traits.ElectricMode_Segment, len(segments)-i)
			out[0] = after
			copy(out[1:], segments[i+1:])
			return out
		}

		cur += l
	}

	return nil
}
