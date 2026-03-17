package modepb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

func TestSum(t *testing.T) {
	tests := []struct {
		name string
		args []*traits.ElectricMode
		want *traits.ElectricMode
	}{
		{"no modes", modes(), nil},
		{"no segments st-none", modes(m(), m()), m()},
		{"no segments st-one", modes(mst(3), m()), mst(3)},
		{"no segments st-all", modes(mst(3), mst(5)), mst(3)},
		{"one mode", modes(m(s{1, 2})), m(s{1, 2})},
		{"one mode st", modes(mst(8, s{1, 2})), mst(8, s{1, 2})},
		{"no st", modes(m(s{1, 10}), m(s{2, 4}, s{3, 2})), m(s{3, 4}, s{4, 2}, s{1, 4})},
		{"one st", modes(m(s{1, 10}), mst(1, s{2, 4}, s{3, 2})), mst(1, s{3, 4}, s{4, 2}, s{1, 4})},
		{"some st", modes(mst(2, s{7, 5}), m(s{1, 10}), mst(1, s{2, 4}, s{3, 2})), mst(1, s{2, 1}, s{10, 3}, s{11, 2}, s{1, 5})},
		{"all st", modes(mst(2, s{7, 5}), mst(3, s{1, 11}), mst(1, s{2, 4}, s{3, 2})), mst(1, s{2, 1}, s{9, 1}, s{10, 2}, s{11, 2}, s{1, 7})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, Sum(tt.args...), protocmp.Transform()); diff != "" {
				t.Error("Sum() (-want, +got)\n" + diff)
			}
		})
	}
}
