package modepb

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	segmentpb2 "github.com/smart-core-os/sc-golang/pkg/trait/electricpb/segmentpb"
)

// Cut cuts the given mode around the time t.
// If mode has no start time then start time is assumed to be t, implying (nil, mode, false) is returned.
// The returned value outside indicates whether t is outside the bounds of mode.
func Cut(t time.Time, mode *traits.ElectricMode) (before, after *traits.ElectricMode, outside bool) {
	if len(mode.GetSegments()) == 0 {
		return mode, mode, true // special case when the mode has no segments
	}

	st := tOrST(t, mode)
	if !t.After(st) { // if t <= st
		return nil, mode, t.Before(st)
	}
	d := t.Sub(st)
	elapsed, index := segmentpb2.ActiveAt(d, mode.Segments...)
	if index == len(mode.Segments) {
		// t is after the mode has ended
		return mode, nil, true
	}

	before = proto.Clone(mode).(*traits.ElectricMode)
	after = proto.Clone(mode).(*traits.ElectricMode)
	after.StartTime = timestamppb.New(t)

	sb, sa, _ := segmentpb2.Cut(d-elapsed, mode.Segments[index])
	if sb == nil {
		before.Segments = before.Segments[:index]
	} else {
		before.Segments = append(before.Segments[:index], sb)
	}
	if sa == nil {
		after.Segments = after.Segments[index+1:]
	} else {
		after.Segments[index] = sa
		after.Segments = after.Segments[index:]
	}

	return before, after, false
}
