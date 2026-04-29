// A collection of test helper functions for this package

package segmentpb

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
)

type s struct {
	m float32
	d time.Duration
}

func seg(seg s) *electricpb.ElectricMode_Segment {
	segment := electricpb.ElectricMode_Segment{Magnitude: seg.m}
	if seg.d != 0 {
		segment.Length = durationpb.New(seg.d)
	}
	return &segment
}

func segs(ss ...s) (result []*electricpb.ElectricMode_Segment) {
	for _, item := range ss {
		result = append(result, seg(item))
	}
	return result
}
