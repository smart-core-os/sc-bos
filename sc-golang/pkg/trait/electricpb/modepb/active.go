package modepb

import (
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/trait/electricpb/segmentpb"
)

// ActiveAt finds the modes active segment at time t.
// The index of the active segment in mode is returned along with the elapsed time before that segment started.
// If mode.Segments is empty, (0,0) is returned.
// If t is before mode starts, index will be 0 and elapsed will be negative, indicating how far before mode.StartTime t is.
// If t is after all segments, the total length of the segments is returned as elapsed, and len(mode.Segments) as index.
// If mode.StartTime is nil, t will be used as the start time, i.e. (0, 0) will be returned.
func ActiveAt(t time.Time, mode *traits.ElectricMode) (elapsed time.Duration, index int) {
	st := tOrST(t, mode)
	d := t.Sub(st)
	return segmentpb.ActiveAt(d, mode.GetSegments()...)
}
