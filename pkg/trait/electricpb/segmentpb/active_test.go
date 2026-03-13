package segmentpb

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestActiveAt(t *testing.T) {
	tests := []struct {
		name        string
		d           time.Duration
		segments    []*traits.ElectricMode_Segment
		wantElapsed time.Duration
		wantIndex   int
	}{
		{"empty", 0, segs(), 0, 0},
		{"negative", -8, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), -8, 0},
		{"start", 0, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 0, 0},
		{"end1", 1, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 1, 1},
		{"mid", 2, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 1, 1},
		{"end2", 3, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 3, 2},
		{"end-n", 11, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 11, 4},
		{"after", 100, segs(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 11, 4},
		{"inf", 3, segs(s{1, 1}, s{2, 2}, s{3, 0}, s{5, 5}), 3, 2},
		{"inf-after", 8, segs(s{1, 1}, s{2, 2}, s{3, 0}, s{5, 5}), 3, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotElapsed, gotIndex := ActiveAt(tt.d, tt.segments...)
			if gotElapsed != tt.wantElapsed {
				t.Errorf("ActiveAt() gotElapsed = %v, want %v", gotElapsed, tt.wantElapsed)
			}
			if gotIndex != tt.wantIndex {
				t.Errorf("ActiveAt() gotIndex = %v, want %v", gotIndex, tt.wantIndex)
			}
		})
	}
}
