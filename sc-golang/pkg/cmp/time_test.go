package cmp

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-golang/internal/testproto"
	"google.golang.org/protobuf/proto"
)

func TestTimeValueWithin(t *testing.T) {
	tests := []struct {
		name string
		cmp  []Value
		x, y proto.Message
		want bool
	}{
		{"(0) t={1,1}", []Value{TimeValueWithin(0)}, tv(1, 0), tv(1, 0), true},
		{"(0) t={1,2}", []Value{TimeValueWithin(0)}, tv(1, 0), tv(2, 0), false},
		{"(0) t={1,1.1}", []Value{TimeValueWithin(0)}, tv(1, 0), tv(1, 1), false},
		{"(1s) t={1,1}", []Value{TimeValueWithin(1 * time.Second)}, tv(1, 0), tv(1, 0), true},
		{"(1s) t={1,1.1}", []Value{TimeValueWithin(1 * time.Second)}, tv(1, 0), tv(1, 1), true},
		{"(1s) t={1,1.999999999}", []Value{TimeValueWithin(1 * time.Second)}, tv(1, 0), tv(1, 999999999), true},
		{"(1s) t={1,2}", []Value{TimeValueWithin(1 * time.Second)}, tv(1, 0), tv(2, 0), true},
		{"(1s) t={1,2.1}", []Value{TimeValueWithin(1 * time.Second)}, tv(1, 0), tv(2, 1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := Equal(tt.cmp...)
			if eq(tt.x, tt.y) != tt.want {
				t.Errorf("%v != %v: x=%v y=%v", tt.name, tt.want, tt.x, tt.y)
			}
			if eq(tt.y, tt.x) != tt.want {
				t.Errorf("%v != %v: (rev) x=%v y=%v", tt.name, tt.want, tt.y, tt.x)
			}
		})
	}
}

func tv(sec, nanos int64) *testproto.TestAllTypes {
	return setWellKnownTimes(allTypesFull(), time.Unix(sec, nanos))
}

func TestDurationValueWithin(t *testing.T) {
	tests := []struct {
		name string
		cmp  []Value
		x, y proto.Message
		want bool
	}{
		{"(0) d={1s,1s}", []Value{DurationValueWithin(0)}, dv(1 * time.Second), dv(1 * time.Second), true},
		{"(0) d={1s,2s}", []Value{DurationValueWithin(0)}, dv(1 * time.Second), dv(2 * time.Second), false},
		{"(0) d={1s,1001ms}", []Value{DurationValueWithin(0)}, dv(1 * time.Second), dv(1001 * time.Millisecond), false},
		{"(1s) d={1s,1s}", []Value{DurationValueWithin(1 * time.Second)}, dv(1 * time.Second), dv(1 * time.Second), true},
		{"(1s) d={1s,1001ms}", []Value{DurationValueWithin(1 * time.Second)}, dv(1 * time.Second), dv(1001 * time.Millisecond), true},
		{"(1s) d={1s,1999ms}", []Value{DurationValueWithin(1 * time.Second)}, dv(1 * time.Second), dv(1999 * time.Millisecond), true},
		{"(1s) d={1s,2s}", []Value{DurationValueWithin(1 * time.Second)}, dv(1 * time.Second), dv(2 * time.Second), true},
		{"(1s) d={1s,2001ms}", []Value{DurationValueWithin(1 * time.Second)}, dv(1 * time.Second), dv(2001 * time.Millisecond), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := Equal(tt.cmp...)
			if eq(tt.x, tt.y) != tt.want {
				t.Errorf("%v != %v: x=%v y=%v", tt.name, tt.want, tt.x, tt.y)
			}
			if eq(tt.y, tt.x) != tt.want {
				t.Errorf("%v != %v: (rev) x=%v y=%v", tt.name, tt.want, tt.y, tt.x)
			}
		})
	}
}

func dv(d time.Duration) *testproto.TestAllTypes {
	return setWellKnownDurations(allTypesFull(), d)
}
