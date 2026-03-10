package cmp

import (
	"testing"

	"github.com/smart-core-os/sc-golang/internal/testproto"
	"google.golang.org/protobuf/proto"
)

func TestFloatValueApprox(t *testing.T) {
	tests := []struct {
		name string
		cmp  []Value
		x, y proto.Message
		want bool
	}{
		{"(0,0) f={1,1}", []Value{FloatValueApprox(0, 0)}, fa(1), fa(1), true},
		{"(0,0) f={1,2}", []Value{FloatValueApprox(0, 0)}, fa(1), fa(2), false},
		{"(0,0) f={1,1.001}", []Value{FloatValueApprox(0, 0)}, fa(1), fa(1.001), false},
		{"(0,1) f={1,1}", []Value{FloatValueApprox(0, 1)}, fa(1), fa(1), true},
		{"(0,1) f={1,1.1}", []Value{FloatValueApprox(0, 1)}, fa(1), fa(1.1), true},
		{"(0,1) f={1,2}", []Value{FloatValueApprox(0, 1)}, fa(1), fa(2), true},
		{"(0,1) f={1,2.00001}", []Value{FloatValueApprox(0, 1)}, fa(1), fa(2.00001), false},
		{"(.1,0) f={100,100}", []Value{FloatValueApprox(.1, 0)}, fa(100), fa(100), true},
		{"(.1,0) f={100,101}", []Value{FloatValueApprox(.1, 0)}, fa(100), fa(101), true},
		{"(.1,0) f={100,110}", []Value{FloatValueApprox(.1, 0)}, fa(100), fa(110), true},
		{"(.1,0) f={100,111}", []Value{FloatValueApprox(.1, 0)}, fa(100), fa(111), false},
		{"(.1,0) f={100,110.0001}", []Value{FloatValueApprox(.1, 0)}, fa(100), fa(110.0001), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := Equal(tt.cmp...)
			if eq(tt.x, tt.y) != tt.want {
				t.Errorf("TimeValueWithin%v != %v: x=%v y=%v", tt.name, tt.want, tt.x, tt.y)
			}
			if eq(tt.y, tt.x) != tt.want {
				t.Errorf("TimeValueWithin%v != %v: (rev) x=%v y=%v", tt.name, tt.want, tt.y, tt.x)
			}
		})
	}
}

func fa(f float64) *testproto.TestAllTypes {
	base := allTypesFull()
	base = setFloatValues(base, float32(f))
	base = setDoubleValues(base, f)
	return base
}
