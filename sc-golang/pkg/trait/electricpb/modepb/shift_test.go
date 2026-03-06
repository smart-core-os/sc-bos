package modepb

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

func TestShift(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		mode *traits.ElectricMode
		want *traits.ElectricMode
	}{
		// tests without start times (copied from segmentpb shift_test)
		{"empty+0", 0, m(), m()},
		{"empty+10", 10, m(), m()},
		{"empty-10", -10, m(), m()},
		{"one+0", 0, m(s{1, 5}), m(s{1, 5})},
		{"one+10", 10, m(s{1, 5}), m(s{0, 10}, s{1, 5})},
		{"one-1 (cut)", -1, m(s{1, 5}), m(s{1, 4})},
		{"one-5 (end-exclusive)", -5, m(s{1, 5}), m()},
		{"one-10 (all)", -10, m(s{1, 5}), m()},
		{"inf+0", 0, m(s{1, 0}), m(s{1, 0})},
		{"inf+10", 10, m(s{1, 0}), m(s{0, 10}, s{1, 0})},
		{"inf-10", -10, m(s{1, 0}), m(s{1, 0})},
		{"few+0", 0, m(s{1, 3}, s{2, 5}), m(s{1, 3}, s{2, 5})},
		{"few+10", 10, m(s{1, 3}, s{2, 5}), m(s{0, 10}, s{1, 3}, s{2, 5})},
		{"few-1 (cut 1st)", -1, m(s{1, 3}, s{2, 5}), m(s{1, 2}, s{2, 5})},
		{"few-3 (end 1st)", -3, m(s{1, 3}, s{2, 5}), m(s{2, 5})},
		{"few-5 (cut 2nd)", -5, m(s{1, 3}, s{2, 5}), m(s{2, 3})},
		{"few-8 (end 2nd)", -8, m(s{1, 3}, s{2, 5}), m()},
		{"few-10 (all)", -10, m(s{1, 3}, s{2, 5}), m()},
		{"few+inf+0", 0, m(s{1, 3}, s{2, 5}, s{3, 0}), m(s{1, 3}, s{2, 5}, s{3, 0})},
		{"few+inf+10", 10, m(s{1, 3}, s{2, 5}, s{3, 0}), m(s{0, 10}, s{1, 3}, s{2, 5}, s{3, 0})},
		{"few+inf-1 (cut 1st)", -1, m(s{1, 3}, s{2, 5}, s{3, 0}), m(s{1, 2}, s{2, 5}, s{3, 0})},
		{"few+inf-3 (end 1st)", -3, m(s{1, 3}, s{2, 5}, s{3, 0}), m(s{2, 5}, s{3, 0})},
		{"few+inf-5 (cut 2nd)", -5, m(s{1, 3}, s{2, 5}, s{3, 0}), m(s{2, 3}, s{3, 0})},
		{"few+inf-8 (end 2nd)", -8, m(s{1, 3}, s{2, 5}, s{3, 0}), m(s{3, 0})},
		{"few+inf-10 (all)", -10, m(s{1, 3}, s{2, 5}, s{3, 0}), m(s{3, 0})},
		// tests with start times
		{"st empty+0", 0, mst(7), mst(7)},
		{"st empty+10", 10, mst(7), mst(17)},
		{"st empty-10", -10, mst(7), mst(-3)},
		{"st one+0", 0, mst(7, s{1, 5}), mst(7, s{1, 5})},
		{"st one+10", 10, mst(7, s{1, 5}), mst(17, s{1, 5})},
		{"st one-10", -10, mst(7, s{1, 5}), mst(-3, s{1, 5})},
		{"st inf+0", 0, mst(7, s{1, 0}), mst(7, s{1, 0})},
		{"st inf+10", 10, mst(7, s{1, 0}), mst(17, s{1, 0})},
		{"st inf-10", -10, mst(7, s{1, 0}), mst(-3, s{1, 0})},
		{"st few+0", 0, mst(7, s{1, 3}, s{2, 5}), mst(7, s{1, 3}, s{2, 5})},
		{"st few+10", 10, mst(7, s{1, 3}, s{2, 5}), mst(17, s{1, 3}, s{2, 5})},
		{"st few-10", -10, mst(7, s{1, 3}, s{2, 5}), mst(-3, s{1, 3}, s{2, 5})},
		{"st few+inf+0", 0, mst(7, s{1, 3}, s{2, 5}, s{3, 0}), mst(7, s{1, 3}, s{2, 5}, s{3, 0})},
		{"st few+inf+10", 10, mst(7, s{1, 3}, s{2, 5}, s{3, 0}), mst(17, s{1, 3}, s{2, 5}, s{3, 0})},
		{"st few+inf-10", -10, mst(7, s{1, 3}, s{2, 5}, s{3, 0}), mst(-3, s{1, 3}, s{2, 5}, s{3, 0})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, Shift(tt.d, tt.mode), protocmp.Transform()); diff != "" {
				t.Error("Shift() (-want, +got)\n" + diff)
			}
		})
	}
}
