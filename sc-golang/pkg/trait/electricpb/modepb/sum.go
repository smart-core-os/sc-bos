package modepb

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/trait/electricpb/segmentpb"
)

// Sum combines all the given modes together into a single mode.
// The magnitudes of each modes segment are added together to produce a new segment array.
// No metadata will be set on the returned mode: id, description, etc.
// Voltage will be ignored and unset in the response.
//
// # Regarding start times
//
// If no modes have a start time, we assume all segments are relative to the same time and will combine them as such.
// If only one mode has a start time, we follow the rules above but also set the start time of the returned mode to this.
// Any mode without a start time is assumed to start at the most recent start time of the modes that do.
// The returned mode will have a start time equal to the earliest start time of the given modes, if one exists.
func Sum(modes ...*traits.ElectricMode) *traits.ElectricMode {
	if len(modes) == 0 {
		return nil
	}
	var earliest, latest time.Time
	var stCount int
	segmentSlices := make([][]*traits.ElectricMode_Segment, len(modes))
	for i, mode := range modes {
		segmentSlices[i] = mode.Segments
		if mode.StartTime != nil {
			stCount++
			st := mode.StartTime.AsTime()
			if earliest.IsZero() || st.Before(earliest) {
				earliest = st
			}
			if latest.IsZero() || st.After(latest) {
				latest = st
			}
		}
	}

	anyHaveST := stCount > 0
	if anyHaveST {
		// Shift the segments around until they align
		for i, mode := range modes {
			st := latest
			if mode.StartTime != nil {
				st = mode.StartTime.AsTime()
			}
			diff := st.Sub(earliest)
			segmentSlices[i] = segmentpb.Shift(diff, segmentSlices[i]...)
		}
	}

	result := &traits.ElectricMode{}
	if anyHaveST {
		result.StartTime = timestamppb.New(earliest)
	}
	result.Segments = segmentpb.Sum(segmentSlices...)

	return result
}
