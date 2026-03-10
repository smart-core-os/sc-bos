package segmentpb

import (
	"reflect"
	"testing"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestCut(t *testing.T) {
	tests := []struct {
		name        string
		d           time.Duration
		segment     *traits.ElectricMode_Segment
		wantBefore  *traits.ElectricMode_Segment
		wantAfter   *traits.ElectricMode_Segment
		wantOutside bool
	}{
		{"-10 inf", -10, seg(s{2, 0}), nil, seg(s{2, 0}), true},
		{"-10 cap", -10, seg(s{2, 10}), nil, seg(s{2, 10}), true},
		{"0 inf", 0, seg(s{2, 0}), nil, seg(s{2, 0}), false},
		{"0 cap", 0, seg(s{2, 10}), nil, seg(s{2, 10}), false},
		{"4 inf", 4, seg(s{2, 0}), seg(s{2, 4}), seg(s{2, 0}), false},
		{"4 cap", 4, seg(s{2, 10}), seg(s{2, 4}), seg(s{2, 6}), false},
		{"10 cap (end)", 10, seg(s{2, 10}), seg(s{2, 10}), nil, true},
		{"20 cap (after)", 20, seg(s{2, 10}), seg(s{2, 10}), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBefore, gotAfter, gotOutside := Cut(tt.d, tt.segment)
			if !reflect.DeepEqual(gotBefore, tt.wantBefore) {
				t.Errorf("Cut() gotBefore = %v, want %v", gotBefore, tt.wantBefore)
			}
			if !reflect.DeepEqual(gotAfter, tt.wantAfter) {
				t.Errorf("Cut() gotAfter = %v, want %v", gotAfter, tt.wantAfter)
			}
			if gotOutside != tt.wantOutside {
				t.Errorf("Cut() gotOutside = %v, want %v", gotOutside, tt.wantOutside)
			}
		})
	}
}
