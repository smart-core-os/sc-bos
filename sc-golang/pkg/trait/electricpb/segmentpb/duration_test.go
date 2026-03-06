package segmentpb

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestDuration(t *testing.T) {
	tests := []struct {
		name         string
		args         []*traits.ElectricMode_Segment
		wantTotal    time.Duration
		wantInfinite bool
	}{
		{"empty", segs(), 0, false},
		{"one", segs(s{0, 10}), 10, false},
		{"inf", segs(s{0, 0}), 0, true},
		{"few", segs(s{0, 1}, s{0, 2}, s{0, 5}), 8, false},
		{"few+inf", segs(s{0, 1}, s{0, 2}, s{0, 5}, s{0, 0}), 8, true},
		{"inf in the middle", segs(s{0, 1}, s{0, 0}, s{0, 5}), 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTotal, gotInfinite := Duration(tt.args...)
			if gotTotal != tt.wantTotal {
				t.Errorf("Duration() gotTotal = %v, want %v", gotTotal, tt.wantTotal)
			}
			if gotInfinite != tt.wantInfinite {
				t.Errorf("Duration() gotInfinite = %v, want %v", gotInfinite, tt.wantInfinite)
			}
		})
	}
}
