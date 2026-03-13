// A collection of test helper functions for this package

package modepb

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

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

func m(ss ...s) *traits.ElectricMode {
	result := &traits.ElectricMode{}
	for _, item := range ss {
		result.Segments = append(result.Segments, seg(item))
	}
	return result
}

func mst(st int, ss ...s) *traits.ElectricMode {
	result := m(ss...)
	result.StartTime = timestamppb.New(at(st))
	return result
}

func modes(modes ...*traits.ElectricMode) []*traits.ElectricMode {
	return modes
}

func at(t int) time.Time {
	return time.Time{}.Add(time.Duration(t))
}
