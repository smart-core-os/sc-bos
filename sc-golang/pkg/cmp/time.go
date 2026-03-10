package cmp

import (
	"time"

	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TimeValueWithin(d time.Duration) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal bool, ok bool) {
		if fd.Kind() != pref.MessageKind {
			return false, false
		}
		mx, my := x.Message(), y.Message()
		xIsTimestamp := mx.Descriptor().FullName() == "google.protobuf.Timestamp"
		yIsTimestamp := my.Descriptor().FullName() == "google.protobuf.Timestamp"
		if !xIsTimestamp && !yIsTimestamp {
			return false, false
		}
		if xIsTimestamp != yIsTimestamp {
			return false, true
		}
		if !mx.IsValid() || !my.IsValid() {
			return mx.IsValid() == my.IsValid(), true
		}

		xt, yt := toTime(mx), toTime(my)
		if xt.Before(yt) {
			return yt.Sub(xt) <= d, true
		}
		return xt.Sub(yt) <= d, true
	}
}

func toTime(x pref.Message) time.Time {
	return x.Interface().(*timestamppb.Timestamp).AsTime()
}

// DurationValueWithin considers two durationpb.Duration to be equal if their durations are within d of each other.
func DurationValueWithin(d time.Duration) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal, ok bool) {
		xd, yd, equal, ok, returnEarly := cmpDuration(fd, x, y)
		if returnEarly {
			return equal, ok
		}
		if xd < yd {
			return yd-xd <= d, true
		}
		return xd-yd <= d, true
	}
}

// DurationValueWithinP considers two durationpb.Duration to be equal if their values are within p percent of each other.
func DurationValueWithinP(p float32) Value {
	return func(fd pref.FieldDescriptor, x, y pref.Value) (equal, ok bool) {
		xd, yd, equal, ok, returnEarly := cmpDuration(fd, x, y)
		if returnEarly {
			return equal, ok
		}
		pd := float32(xd) / float32(yd)
		if pd < 0 {
			pd = -pd
		}
		return pd < p, true
	}
}

func cmpDuration(fd pref.FieldDescriptor, x, y pref.Value) (xd, yd time.Duration, equal, ok, returnEarly bool) {
	if fd.Kind() != pref.MessageKind {
		return xd, yd, false, false, true
	}
	mx, my := x.Message(), y.Message()
	xIsDuration := mx.Descriptor().FullName() == "google.protobuf.Duration"
	yIsDuration := my.Descriptor().FullName() == "google.protobuf.Duration"
	if !xIsDuration && !yIsDuration {
		return xd, yd, false, false, true
	}
	if xIsDuration != yIsDuration {
		return xd, yd, false, true, true
	}
	if !mx.IsValid() || !my.IsValid() {
		return xd, yd, mx.IsValid() == my.IsValid(), true, true
	}

	xd, yd = toDuration(mx), toDuration(my)
	return xd, yd, false, false, false
}

func toDuration(x pref.Message) time.Duration {
	return x.Interface().(*durationpb.Duration).AsDuration()
}
