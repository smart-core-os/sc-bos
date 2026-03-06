package modepb

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestActiveAt(t *testing.T) {
	tests := []struct {
		name        string
		t           time.Time
		mode        *traits.ElectricMode
		wantElapsed time.Duration
		wantIndex   int
	}{
		// no start times, all return 0, 0
		{"empty", at(0), m(), 0, 0},
		{"no-st1", at(0), m(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 0, 0},
		{"no-st2", at(1000), m(s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 0, 0},
		{"before", at(0), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), -5, 0},
		{"before2", at(2), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), -3, 0},
		{"start", at(5), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 0, 0},
		{"second start", at(6), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 1, 1},
		{"third inside", at(9), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 3, 2},
		{"third inside2", at(10), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 3, 2},
		{"end", at(16), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 11, 4},
		{"after", at(100), mst(5, s{1, 1}, s{2, 2}, s{3, 3}, s{5, 5}), 11, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotElapsed, gotIndex := ActiveAt(tt.t, tt.mode)
			if gotElapsed != tt.wantElapsed {
				t.Errorf("ActiveAt() gotElapsed = %v, want %v", gotElapsed, tt.wantElapsed)
			}
			if gotIndex != tt.wantIndex {
				t.Errorf("ActiveAt() gotIndex = %v, want %v", gotIndex, tt.wantIndex)
			}
		})
	}
}
