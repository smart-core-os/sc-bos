package cmp

import (
	"math"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// FloatValueApprox compares floating point values as equal if they are within
// fraction or margin of each other.
func FloatValueApprox(fraction, margin float64) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal bool, ok bool) {
		if fd.Kind() != pref.FloatKind && fd.Kind() != pref.DoubleKind {
			return false, false
		}
		fx, fy := x.Float(), y.Float()
		relMarg := fraction * math.Min(math.Abs(fx), math.Abs(fy))
		return math.Abs(fx-fy) <= math.Max(margin, relMarg), true
	}
}
