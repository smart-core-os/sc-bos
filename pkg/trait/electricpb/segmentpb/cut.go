package segmentpb

import (
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Cut divides a segment in two along d.
// Cutting an infinite segment results in the same infinite segment for both before and after.
// If d is negative, `nil, segment` is returned.
func Cut(d time.Duration, segment *traits.ElectricMode_Segment) (before, after *traits.ElectricMode_Segment, outside bool) {
	if d <= 0 {
		return nil, segment, d < 0
	}
	if segment.GetLength() == nil {
		return &traits.ElectricMode_Segment{
			Magnitude: segment.GetMagnitude(),
			Length:    durationpb.New(d),
		}, segment, false
	}

	l := segment.GetLength().AsDuration()
	if l <= d {
		return segment, nil, true
	}

	before = &traits.ElectricMode_Segment{
		Magnitude: segment.GetMagnitude(),
		Length:    durationpb.New(d),
	}
	after = &traits.ElectricMode_Segment{
		Magnitude: segment.GetMagnitude(),
		Length:    durationpb.New(l - d),
	}

	// handle shapes
	if segment.GetShape() != nil {
		switch shape := segment.GetShape().(type) {
		case *traits.ElectricMode_Segment_Fixed:
			before.Shape = &traits.ElectricMode_Segment_Fixed{Fixed: shape.Fixed}
			after.Shape = &traits.ElectricMode_Segment_Fixed{Fixed: shape.Fixed}
		}
	}

	return before, after, false
}
