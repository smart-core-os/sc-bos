package segmentpb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

func TestSum(t *testing.T) {
	tests := []struct {
		name string
		args [][]*traits.ElectricMode_Segment
		want []*traits.ElectricMode_Segment
	}{
		{"[] empty", arg(), segs()},
		{"[[{10,}]] one-inf", arg(segs(s{10, 0})), segs(s{10, 0})},
		{"[[{10,10}]] one-cap", arg(segs(s{10, 10})), segs(s{10, 10})},
		{"[[{10,10},{15,}]] two-inf", arg(segs(s{10, 10}, s{15, 0})), segs(s{10, 10}, s{15, 0})},
		{"[[{10,10},{5,15}]] two-cap", arg(segs(s{10, 10}, s{5, 15})), segs(s{10, 10}, s{5, 15})},
		{"[[{10,10}),{5,15}]] one-cap,one-cap", arg(segs(s{10, 10}), segs(s{5, 15})), segs(s{15, 10}, s{5, 5})},
		{"[[{10,}),{5,15}]] one-inf,one-cap", arg(segs(s{10, 0}), segs(s{5, 15})), segs(s{15, 15}, s{10, 0})},
		{"[[{10,20}],[{0,5},{5,10}]] nested", arg(segs(s{10, 20}), segs(s{0, 5}, s{5, 10})), segs(s{10, 5}, s{15, 10}, s{10, 5})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, Sum(tt.args...), protocmp.Transform()); diff != "" {
				t.Error("Sum() (-want, +got) \n" + diff)
			}
		})
	}
}

func Test_calcCuts(t *testing.T) {
	tests := []struct {
		name string
		args [][]*traits.ElectricMode_Segment
		want []cut
	}{
		{"[] empty", arg(), cuts()},
		{"[[{10,}]] one-inf", arg(segs(s{10, 0})), cuts(cut{0, 10})},
		{"[[{10,10}]] one-cap", arg(segs(s{10, 10})), cuts(cut{0, 10}, cut{10, -10})},
		{"[[{10,10},{15,}]] two-inf", arg(segs(s{10, 10}, s{15, 0})), cuts(cut{0, 10}, cut{10, -10}, cut{10, 15})},
		{"[[{10,10},{5,15}]] two-cap", arg(segs(s{10, 10}, s{5, 15})), cuts(cut{0, 10}, cut{10, -10}, cut{10, 5}, cut{25, -5})},
		{"[[{10,10}],[{5,15}]] one-cap,one-cap", arg(segs(s{10, 10}), segs(s{5, 15})), cuts(cut{0, 10}, cut{0, 5}, cut{10, -10}, cut{15, -5})},
		{"[[{10,}],[{5,15}]] one-inf,one-cap", arg(segs(s{10, 0}), segs(s{5, 15})), cuts(cut{0, 10}, cut{0, 5}, cut{15, -5})},
		{"[[{10,20}],[{0,5},{5,10}]] nested", arg(segs(s{10, 20}), segs(s{0, 5}, s{5, 10})), cuts(cut{0, 10}, cut{5, 5}, cut{15, -5}, cut{20, -10})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, calcCuts(tt.args...), cmp.Comparer(compareCuts)); diff != "" {
				t.Error("calcCuts() (-want, +got) \n" + diff)
			}
		})
	}
}

func arg(ss ...[]*traits.ElectricMode_Segment) [][]*traits.ElectricMode_Segment {
	return ss
}

func cuts(cs ...cut) []cut {
	return cs
}

func compareCuts(a, b cut) bool {
	return a == b
}
