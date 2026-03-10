package modepb

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestMaxSegmentAfter(t *testing.T) {
	tests := []struct {
		name      string
		t         time.Time
		mode      *traits.ElectricMode
		wantIndex int
	}{
		// Without start time (same as segmentpb.TestMaxAfter)
		{"empty+0", at(0), m(), 0},
		{"empty+10", at(10), m(), 0},
		{"empty-10", at(-10), m(), 0},
		{"inf+0", at(0), m(s{3, 0}), 0},
		{"inf+10", at(10), m(s{3, 0}), 0},
		{"inf-10", at(-10), m(s{3, 0}), 0},
		{"cap+0", at(0), m(s{3, 7}), 0},
		{"cap+100", at(7), m(s{3, 7}), 0},
		{"cap-10", at(-10), m(s{3, 7}), 0},
		{"max-1", at(-1), m(s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"max+0", at(0), m(s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"max+1", at(1), m(s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"max+10", at(10), m(s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"max+1000", at(1000), m(s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"max+inf+8", at(8), m(s{3, 3}, s{5, 5}, s{2, 0}), 1},
		{"max+inf+10", at(10), m(s{3, 3}, s{5, 5}, s{2, 0}), 1},
		{"max+inf+1000", at(1000), m(s{3, 3}, s{5, 5}, s{2, 0}), 1},
		// With start time
		{"st empty+0", at(0), mst(5), 0},
		{"st empty+10", at(10), mst(5), 0},
		{"st empty-10", at(-10), mst(5), 0},
		{"st inf+0", at(0), mst(5, s{3, 0}), 0},
		{"st inf+5", at(10), mst(5, s{3, 0}), 0},
		{"st inf+10", at(10), mst(5, s{3, 0}), 0},
		{"st inf-10", at(-10), mst(5, s{3, 0}), 0},
		{"st cap+0", at(0), mst(5, s{3, 7}), 0},
		{"st cap+1", at(1), mst(5, s{3, 7}), 0},
		{"st cap+5", at(5), mst(5, s{3, 7}), 0},
		{"st cap+6", at(6), mst(5, s{3, 7}), 0},
		{"st cap+12", at(12), mst(5, s{3, 7}), 1},
		{"st cap-10", at(-10), mst(5, s{3, 7}), 0},
		{"st max-1", at(-1), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"st max+0", at(0), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"st max+1", at(1), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"st max+5", at(5), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"st max+6", at(6), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"st max+8", at(8), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 1},
		{"st max+13", at(13), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 2},
		{"st max+15", at(15), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 3},
		{"st max+150", at(150), mst(5, s{3, 3}, s{5, 5}, s{2, 2}), 3},
		{"st max+inf+13", at(13), mst(5, s{3, 3}, s{5, 5}, s{2, 0}), 2},
		{"st max+inf+20", at(20), mst(5, s{3, 3}, s{5, 5}, s{2, 0}), 2},
		{"st max+inf+1000", at(1000), mst(5, s{3, 3}, s{5, 5}, s{2, 0}), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotIndex := MaxSegmentAfter(tt.t, tt.mode); gotIndex != tt.wantIndex {
				t.Errorf("MaxSegmentAfter() = %v, want %v", gotIndex, tt.wantIndex)
			}
		})
	}
}
