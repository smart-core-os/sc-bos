package modepb

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
)

func TestCut(t *testing.T) {
	tests := []struct {
		name        string
		t           time.Time
		mode        *traits.ElectricMode
		wantBefore  *traits.ElectricMode
		wantAfter   *traits.ElectricMode
		wantOutside bool
	}{
		// no start time
		{"inf", at(10), m(s{2, 0}), nil, m(s{2, 0}), false},
		{"cap", at(-10), m(s{2, 10}), nil, m(s{2, 10}), false},
		{"some", at(100), m(s{2, 10}, s{3, 11}), nil, m(s{2, 10}, s{3, 11}), false},
		// has a start time
		{"st before", at(1), mst(2, s{2, 2}, s{3, 3}), nil, mst(2, s{2, 2}, s{3, 3}), true},
		{"st start", at(2), mst(2, s{2, 2}, s{3, 3}), nil, mst(2, s{2, 2}, s{3, 3}), false},
		{"st end", at(7), mst(2, s{2, 2}, s{3, 3}), mst(2, s{2, 2}, s{3, 3}), nil, true},
		{"st after", at(10), mst(2, s{2, 2}, s{3, 3}), mst(2, s{2, 2}, s{3, 3}), nil, true},
		{"st cut first", at(3), mst(2, s{2, 2}, s{3, 3}), mst(2, s{2, 1}), mst(3, s{2, 1}, s{3, 3}), false},
		{"st cut between", at(4), mst(2, s{2, 2}, s{3, 3}), mst(2, s{2, 2}), mst(4, s{3, 3}), false},
		{"st cut second", at(5), mst(2, s{2, 2}, s{3, 3}), mst(2, s{2, 2}, s{3, 1}), mst(5, s{3, 2}), false},
		{"st cut inf", at(10), mst(2, s{2, 2}, s{3, 0}), mst(2, s{2, 2}, s{3, 6}), mst(10, s{3, 0}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBefore, gotAfter, gotOutside := Cut(tt.t, tt.mode)
			if diff := cmp.Diff(tt.wantBefore, gotBefore, protocmp.Transform()); diff != "" {
				t.Errorf("Cut() gotBefore (-want,+got)\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantAfter, gotAfter, protocmp.Transform()); diff != "" {
				t.Errorf("Cut() gotAfter (-want,+got)\n%s", diff)
			}
			if gotOutside != tt.wantOutside {
				t.Errorf("Cut() gotOutside = %v, want %v", gotOutside, tt.wantOutside)
			}
		})
	}
}
