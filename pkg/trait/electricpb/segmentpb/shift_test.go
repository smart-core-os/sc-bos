package segmentpb

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
)

func TestShift(t *testing.T) {
	tests := []struct {
		name     string
		d        time.Duration
		segments []*electricpb.ElectricMode_Segment
		want     []*electricpb.ElectricMode_Segment
	}{
		{"empty+0", 0, segs(), segs()},
		{"empty+10", 10, segs(), segs()},
		{"empty-10", -10, segs(), segs()},
		{"one+0", 0, segs(s{1, 5}), segs(s{1, 5})},
		{"one+10", 10, segs(s{1, 5}), segs(s{0, 10}, s{1, 5})},
		{"one-1 (cut)", -1, segs(s{1, 5}), segs(s{1, 4})},
		{"one-5 (end-exclusive)", -5, segs(s{1, 5}), segs()},
		{"one-10 (all)", -10, segs(s{1, 5}), segs()},
		{"inf+0", 0, segs(s{1, 0}), segs(s{1, 0})},
		{"inf+10", 10, segs(s{1, 0}), segs(s{0, 10}, s{1, 0})},
		{"inf-10", -10, segs(s{1, 0}), segs(s{1, 0})},
		{"few+0", 0, segs(s{1, 3}, s{2, 5}), segs(s{1, 3}, s{2, 5})},
		{"few+10", 10, segs(s{1, 3}, s{2, 5}), segs(s{0, 10}, s{1, 3}, s{2, 5})},
		{"few-1 (cut 1st)", -1, segs(s{1, 3}, s{2, 5}), segs(s{1, 2}, s{2, 5})},
		{"few-3 (end 1st)", -3, segs(s{1, 3}, s{2, 5}), segs(s{2, 5})},
		{"few-5 (cut 2nd)", -5, segs(s{1, 3}, s{2, 5}), segs(s{2, 3})},
		{"few-8 (end 2nd)", -8, segs(s{1, 3}, s{2, 5}), segs()},
		{"few-10 (all)", -10, segs(s{1, 3}, s{2, 5}), segs()},
		{"few+inf+0", 0, segs(s{1, 3}, s{2, 5}, s{3, 0}), segs(s{1, 3}, s{2, 5}, s{3, 0})},
		{"few+inf+10", 10, segs(s{1, 3}, s{2, 5}, s{3, 0}), segs(s{0, 10}, s{1, 3}, s{2, 5}, s{3, 0})},
		{"few+inf-1 (cut 1st)", -1, segs(s{1, 3}, s{2, 5}, s{3, 0}), segs(s{1, 2}, s{2, 5}, s{3, 0})},
		{"few+inf-3 (end 1st)", -3, segs(s{1, 3}, s{2, 5}, s{3, 0}), segs(s{2, 5}, s{3, 0})},
		{"few+inf-5 (cut 2nd)", -5, segs(s{1, 3}, s{2, 5}, s{3, 0}), segs(s{2, 3}, s{3, 0})},
		{"few+inf-8 (end 2nd)", -8, segs(s{1, 3}, s{2, 5}, s{3, 0}), segs(s{3, 0})},
		{"few+inf-10 (all)", -10, segs(s{1, 3}, s{2, 5}, s{3, 0}), segs(s{3, 0})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, Shift(tt.d, tt.segments...), protocmp.Transform()); diff != "" {
				t.Error("Shift() (-want, +got)\n" + diff)
			}
		})
	}
}
