package bookingpb

import (
	"math"
	"testing"
)

// A negative page size used to slice before the start of the list and panic (SCB-1470).
func TestCapPageSize(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, defaultPageSize},
		{-1, defaultPageSize},
		{math.MinInt32, defaultPageSize},
		{10, 10},
		{maxPageSize + 1, maxPageSize},
	}
	for _, tt := range tests {
		if got := capPageSize(tt.in); got != tt.want {
			t.Errorf("capPageSize(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
