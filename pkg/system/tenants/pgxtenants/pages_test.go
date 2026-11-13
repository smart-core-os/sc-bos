package pgxtenants

import (
	"math"
	"slices"
	"testing"
)

// A negative page size used to index before the start of the list and panic (SCB-1470).
func TestPaginate_NegativePageSize(t *testing.T) {
	for _, size := range []int32{-1, math.MinInt32} {
		page, next, total, err := paginate([]string{"b", "a"}, size, "", func(s string) string { return s })
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if !slices.Equal(page, []string{"a", "b"}) || total != 2 || next != "" {
			t.Errorf("size %d: got page %v, total %d, next %q; want the default page of both items", size, page, total, next)
		}
	}
}
