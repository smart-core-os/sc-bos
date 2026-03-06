package cmp

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-golang/internal/testproto"
	"google.golang.org/protobuf/proto"
)

func TestValueAnd(t *testing.T) {
	tests := []struct {
		name string
		arg  Value
		x, y proto.Message
		want bool
	}{
		// these should be the same as Equal
		{"[] {t,f}={1.0,1},{1.0,1}", ValueAnd(), tf(1, 0, 1), tf(1, 0, 1), true},
		{"[] {t,f}={1.0,1},{1.1,1}", ValueAnd(), tf(1, 0, 1), tf(1, 1, 1), false},
		{"[] {t,f}={1.0,1},{1.0,1.1}", ValueAnd(), tf(1, 0, 1), tf(1, 0, 1.1), false},
		{"[f(0,.2)] {t,f}={1.0,1},{1.0,1.1}", ValueAnd(FloatValueApprox(0, .2)), tf(1, 0, 1), tf(1, 0, 1.1), true},
		{"[t(1s)] {t,f}={1.0,1},{2.0,1}", ValueAnd(TimeValueWithin(1 * time.Second)), tf(1, 0, 1), tf(2, 0, 1), true},
		{"[t(1s),f(0,.2)] {t,f}={1.0,1},{2.0,1.1}", ValueAnd(TimeValueWithin(1*time.Second), FloatValueApprox(0, .2)), tf(1, 0, 1), tf(2, 0, 1.1), true},
		{"[t(1s),f(0,.2)] {t,f}={1.0,1},{2.0,1}", ValueAnd(TimeValueWithin(1*time.Second), FloatValueApprox(0, .2)), tf(1, 0, 1), tf(2, 0, 1), true},
		{"[t(1s),f(0,.2)] {t,f}={1.0,1},{1.0,1.1}", ValueAnd(TimeValueWithin(1*time.Second), FloatValueApprox(0, .2)), tf(1, 0, 1), tf(1, 0, 1.1), true},
		{"[t(1s),f(0,.2)] {t,f}={1.0,1},{1.100,1.1}", ValueAnd(TimeValueWithin(1*time.Second), FloatValueApprox(0, .2)), tf(1, 0, 1), tf(1, 100, 1.1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := Equal(tt.arg)
			if eq(tt.x, tt.y) != tt.want {
				t.Errorf("!%v: x=%v y=%v", tt.want, tt.x, tt.y)
			}
			if eq(tt.y, tt.x) != tt.want {
				t.Errorf("!%v: (rev) x=%v y=%v", tt.want, tt.y, tt.x)
			}
		})
	}
}

func tf(sec, nsec int64, f float64) *testproto.TestAllTypes {
	base := allTypesFull()
	base = setFloatValues(base, float32(f))
	base = setDoubleValues(base, f)
	base = setWellKnownTimes(base, time.Unix(sec, nsec))
	return base
}
