package segmentpb

import (
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-api/go/traits"
)

// Sum combines the segment lists given by segmentSlices together into a single segment list.
func Sum(segmentSlices ...[]*traits.ElectricMode_Segment) []*traits.ElectricMode_Segment {
	// We ignore shape for now, that's a little hard to implement

	// cuts stores all the rising and falling edges of the segment slices
	cuts := calcCuts(segmentSlices...)

	var result []*traits.ElectricMode_Segment
	var lastTime time.Duration
	for _, cut := range cuts {
		length := cut.at - lastTime

		if len(result) == 0 {
			result = append(result, &traits.ElectricMode_Segment{})
		}

		if length == 0 {
			result[len(result)-1].Magnitude += cut.delta
			continue
		}

		lastResult := result[len(result)-1]
		lastResult.Length = durationpb.New(length)
		result = append(result, &traits.ElectricMode_Segment{Magnitude: lastResult.Magnitude + cut.delta})

		lastTime = cut.at
	}

	if len(result) > 0 {
		last := result[len(result)-1]
		if last.Length == nil && last.Magnitude <= 0 {
			result = result[:len(result)-1]
		}
	}

	return result
}

func calcCuts(segmentSlices ...[]*traits.ElectricMode_Segment) []cut {
	var cuts []cut
	for _, slice := range segmentSlices {
		var cur time.Duration
		for _, segment := range slice {
			delta := segment.Magnitude
			if delta != 0 {
				// only add a cut if the magnitude changes
				cuts = append(cuts, cut{at: cur, delta: delta})
			}
			if segment.Length == nil {
				break
			}
			cur += segment.Length.AsDuration()
			if delta != 0 {
				cuts = append(cuts, cut{at: cur, delta: -delta})
			}
		}
	}

	// sort by time
	sort.Slice(cuts, func(i, j int) bool {
		return cuts[i].at < cuts[j].at
	})
	return cuts
}

type cut struct {
	at    time.Duration
	delta float32
}
