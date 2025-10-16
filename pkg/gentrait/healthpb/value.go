package healthpb

import (
	"reflect"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vanti-dev/sc-bos/pkg/gen"
)

// BoolValue creates a bool HealthCheck_Value.
func BoolValue(b bool) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_BoolValue{BoolValue: b}}
}

// StringValue creates a string HealthCheck_Value.
func StringValue(s string) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_StringValue{StringValue: s}}
}

// IntValue creates an int HealthCheck_Value.
func IntValue(i int64) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_IntValue{IntValue: i}}
}

// UintValue creates a uint HealthCheck_Value.
func UintValue(u uint64) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_UintValue{UintValue: u}}
}

// FloatValue creates a float HealthCheck_Value.
func FloatValue(f float64) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_FloatValue{FloatValue: f}}
}

// TimestampValue creates a timestamp HealthCheck_Value.
func TimestampValue(t time.Time) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_TimestampValue{TimestampValue: timestamppb.New(t)}}
}

// DurationValue creates a duration HealthCheck_Value.
func DurationValue(d time.Duration) *gen.HealthCheck_Value {
	return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_DurationValue{DurationValue: durationpb.New(d)}}
}

// SameValueType returns true if all values have the same underlying type.
func SameValueType(vals ...*gen.HealthCheck_Value) bool {
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
func AddValues(val, delta *gen.HealthCheck_Value) *gen.HealthCheck_Value {
	if val == nil || delta == nil {
		return val
	}
	switch v := val.GetValue().(type) {
	case *gen.HealthCheck_Value_IntValue:
		return IntValue(v.IntValue + delta.GetIntValue())
	case *gen.HealthCheck_Value_UintValue:
		return UintValue(v.UintValue + delta.GetUintValue())
	case *gen.HealthCheck_Value_FloatValue:
		return FloatValue(v.FloatValue + delta.GetFloatValue())
	case *gen.HealthCheck_Value_TimestampValue:
		return TimestampValue(v.TimestampValue.AsTime().Add(delta.GetDurationValue().AsDuration()))
	case *gen.HealthCheck_Value_DurationValue:
		return DurationValue(v.DurationValue.AsDuration() + delta.GetDurationValue().AsDuration())
	}
	return val
}
