package modepb

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/trait/electricpb/segmentpb"
)

// Shift adjusts the given mode until it's start time is offset by d.
// Where a start time exists, this simply changes the start time of the mode by d.
// If no start time exists, this is equivalent to calling segmentpb.Shift on the mode segments.
//
// If d is not 0, a new mode instance will be returned.
func Shift(d time.Duration, mode *traits.ElectricMode) *traits.ElectricMode {
	if d == 0 {
		return mode
	}
	mode = proto.Clone(mode).(*traits.ElectricMode)
	if mode.StartTime == nil {
		mode.Segments = segmentpb.Shift(d, mode.Segments...)
	} else {
		mode.StartTime = timestamppb.New(mode.StartTime.AsTime().Add(d))
	}
	return mode
}
