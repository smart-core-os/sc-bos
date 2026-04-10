package segmentpb

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
)

// Cut divides a segment in two along d.
// Cutting an infinite segment results in the same infinite segment for both before and after.
// If d is negative, `nil, segment` is returned.
func Cut(d time.Duration, segment *electricpb.ElectricMode_Segment) (before, after *electricpb.ElectricMode_Segment, outside bool) {
	if d <= 0 {
		return nil, segment, d < 0
	}
	if segment.GetLength() == nil {
		return &electricpb.ElectricMode_Segment{
			Magnitude: segment.GetMagnitude(),
			Length:    durationpb.New(d),
		}, segment, false
	}

	l := segment.GetLength().AsDuration()
	if l <= d {
		return segment, nil, true
	}

	before = &electricpb.ElectricMode_Segment{
		Magnitude: segment.GetMagnitude(),
		Length:    durationpb.New(d),
	}
	after = &electricpb.ElectricMode_Segment{
		Magnitude: segment.GetMagnitude(),
		Length:    durationpb.New(l - d),
	}

	// handle shapes
	if segment.GetShape() != nil {
		switch shape := segment.GetShape().(type) {
		case *electricpb.ElectricMode_Segment_Fixed:
			before.Shape = &electricpb.ElectricMode_Segment_Fixed{Fixed: shape.Fixed}
			after.Shape = &electricpb.ElectricMode_Segment_Fixed{Fixed: shape.Fixed}
		}
	}

	return before, after, false
}
