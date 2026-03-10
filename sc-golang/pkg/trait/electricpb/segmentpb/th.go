// A collection of test helper functions for this package

package segmentpb

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-api/go/traits"
)

type s struct {
	m float32
	d time.Duration
}

func seg(seg s) *traits.ElectricMode_Segment {
	segment := traits.ElectricMode_Segment{Magnitude: seg.m}
	if seg.d != 0 {
		segment.Length = durationpb.New(seg.d)
	}
	return &segment
}

func segs(ss ...s) (result []*traits.ElectricMode_Segment) {
	for _, item := range ss {
		result = append(result, seg(item))
	}
	return result
}
