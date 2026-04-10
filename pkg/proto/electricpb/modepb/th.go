// A collection of test helper functions for this package

package modepb

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

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

func m(ss ...s) *electricpb.ElectricMode {
	result := &electricpb.ElectricMode{}
	for _, item := range ss {
		result.Segments = append(result.Segments, seg(item))
	}
	return result
}

func mst(st int, ss ...s) *electricpb.ElectricMode {
	result := m(ss...)
	result.StartTime = timestamppb.New(at(st))
	return result
}

func modes(modes ...*electricpb.ElectricMode) []*electricpb.ElectricMode {
	return modes
}

func at(t int) time.Time {
	return time.Time{}.Add(time.Duration(t))
}
