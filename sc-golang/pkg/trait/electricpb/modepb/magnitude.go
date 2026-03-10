package modepb

import (
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/trait/electricpb/segmentpb"
)

// MagnitudeAt returns the magnitude of the mode at t.
// If there is no segment at t, ok will be false.
// If mode does not have a StartTime, returns the magnitude of the first segment.
func MagnitudeAt(t time.Time, mode *traits.ElectricMode) (level float32, ok bool) {
	return segmentpb.MagnitudeAt(t.Sub(tOrST(t, mode)), mode.GetSegments()...)
}

// MinAt returns the mode with the smalled magnitude at the given time, and that magnitude.
// The returned mode will be nil if modes is empty.
// If a mode exists without a segment at time t, it's magnitude is treated as 0 and is a candidate to be returned.
func MinAt(t time.Time, modes map[string]*traits.ElectricMode) (mode *traits.ElectricMode, magnitude float32) {
	for _, electricMode := range modes {
		electricMode := electricMode
		mag, _ := MagnitudeAt(t, electricMode)
		if mode == nil || mag < magnitude {
			mode = electricMode
			magnitude = mag
		}
	}
	return
}

// MaxSegmentAfter returns the index in mode.Segments of the segment with the largest magnitude after the given time.
// If mode.StartTime is nil, this function is equivalent to segmentpb.Max(mode.Segments), i.e. mode.StartTime == t.
func MaxSegmentAfter(t time.Time, mode *traits.ElectricMode) (index int) {
	return segmentpb.MaxAfter(t.Sub(tOrST(t, mode)), mode.GetSegments()...)
}
