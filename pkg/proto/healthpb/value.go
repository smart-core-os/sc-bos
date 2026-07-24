package healthpb

import (
	"reflect"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BoolValue creates a bool HealthCheck_Value.
func BoolValue(b bool) *HealthCheck_Value {
	return &HealthCheck_Value{Value: &HealthCheck_Value_BoolValue{BoolValue: b}}
}

// StringValue creates a string HealthCheck_Value.
func StringValue(s string) *HealthCheck_Value {
	return &HealthCheck_Value{Value: &HealthCheck_Value_StringValue{StringValue: s}}
}

// IntValue creates an int HealthCheck_Value.
func IntValue(i int64) *HealthCheck_Value {
	return &HealthCheck_Value{Value: &HealthCheck_Value_IntValue{IntValue: i}}
}

// UintValue creates a uint HealthCheck_Value.
func UintValue(u uint64) *HealthCheck_Value {
	return &HealthCheck_Value{Value: &HealthCheck_Value_UintValue{UintValue: u}}
}

// FloatValue creates a float HealthCheck_Value.
func FloatValue(f float64) *HealthCheck_Value {
	return &HealthCheck_Value{Value: &HealthCheck_Value_FloatValue{FloatValue: f}}
}

// TimestampValue creates a timestamp HealthCheck_Value.
func TimestampValue(t time.Time) *HealthCheck_Value {
	return &HealthCheck_Value{Value: &HealthCheck_Value_TimestampValue{TimestampValue: timestamppb.New(t)}}
}

// DurationValue creates a duration HealthCheck_Value.
func DurationValue(d time.Duration) *HealthCheck_Value {
	return &HealthCheck_Value{Value: &HealthCheck_Value_DurationValue{DurationValue: durationpb.New(d)}}
}

// SameValueType returns true if all values have the same underlying type.
func SameValueType(vals ...*HealthCheck_Value) bool {
	if len(vals) < 2 {
		return true
	}
	t := reflect.TypeOf(vals[0].GetValue())
	for _, v := range vals[1:] {
		if reflect.TypeOf(v.GetValue()) != t {
			return false
		}
	}
	return true
}

// AddValues add delta to val, returning a new value.
// If the underlying types for both val and delta are not numeric and the same, val is returned unchanged.
func AddValues(val, delta *HealthCheck_Value) *HealthCheck_Value {
	if val == nil || delta == nil {
		return val
	}
	switch v := val.GetValue().(type) {
	case *HealthCheck_Value_IntValue:
		return IntValue(v.IntValue + delta.GetIntValue())
	case *HealthCheck_Value_UintValue:
		return UintValue(v.UintValue + delta.GetUintValue())
	case *HealthCheck_Value_FloatValue:
		return FloatValue(v.FloatValue + delta.GetFloatValue())
	case *HealthCheck_Value_TimestampValue:
		return TimestampValue(v.TimestampValue.AsTime().Add(delta.GetDurationValue().AsDuration()))
	case *HealthCheck_Value_DurationValue:
		return DurationValue(v.DurationValue.AsDuration() + delta.GetDurationValue().AsDuration())
	}
	return val
}

// valueAsFloat converts an ordered value to a float64 for magnitude calculations.
// It returns ok=false for bool and string values, which have no meaningful
// numeric magnitude. Timestamps use Unix nanoseconds; durations use nanoseconds.
func valueAsFloat(v *HealthCheck_Value) (float64, bool) {
	switch x := v.GetValue().(type) {
	case *HealthCheck_Value_IntValue:
		return float64(x.IntValue), true
	case *HealthCheck_Value_UintValue:
		return float64(x.UintValue), true
	case *HealthCheck_Value_FloatValue:
		return x.FloatValue, true
	case *HealthCheck_Value_TimestampValue:
		return float64(x.TimestampValue.AsTime().UnixNano()), true
	case *HealthCheck_Value_DurationValue:
		return float64(x.DurationValue.AsDuration().Nanoseconds()), true
	}
	return 0, false
}

// isTimestampValue reports whether v holds a timestamp. Timestamps are measured
// from an arbitrary epoch, so their absolute magnitude is not a meaningful scale.
func isTimestampValue(v *HealthCheck_Value) bool {
	_, ok := v.GetValue().(*HealthCheck_Value_TimestampValue)
	return ok
}
