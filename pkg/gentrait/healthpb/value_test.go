package healthpb

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/vanti-dev/sc-bos/pkg/gen"
)

func TestSameValueType(t *testing.T) {
	tests := []struct {
		name string
		vals []*gen.HealthCheck_Value
		want bool
	}{
		{"empty", nil, true},
		{"one", []*gen.HealthCheck_Value{IntValue(1)}, true},
		{"many", []*gen.HealthCheck_Value{IntValue(1), IntValue(2), IntValue(3)}, true},
		{"int", []*gen.HealthCheck_Value{IntValue(1), IntValue(2), IntValue(3)}, true},
		{"uint", []*gen.HealthCheck_Value{UintValue(1), UintValue(2), UintValue(3)}, true},
		{"float", []*gen.HealthCheck_Value{FloatValue(1.5), FloatValue(2.5), FloatValue(3.5)}, true},
		{"bool", []*gen.HealthCheck_Value{BoolValue(true), BoolValue(false), BoolValue(true)}, true},
		{"string", []*gen.HealthCheck_Value{StringValue("a"), StringValue("b"), StringValue("c")}, true},
		{"timestamp", []*gen.HealthCheck_Value{TimestampValue(time.Now()), TimestampValue(time.Now()), TimestampValue(time.Now())}, true},
		{"duration", []*gen.HealthCheck_Value{DurationValue(1 * time.Second), DurationValue(2 * time.Second), DurationValue(3 * time.Second)}, true},
		// mixed types
		{"i,f,i", []*gen.HealthCheck_Value{IntValue(1), FloatValue(2.5), IntValue(3)}, false},
		{"t,d", []*gen.HealthCheck_Value{TimestampValue(time.Now()), DurationValue(2 * time.Second)}, false},
		{"s,s,i", []*gen.HealthCheck_Value{StringValue("a"), StringValue("b"), IntValue(3)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SameValueType(tt.vals...)
			if got != tt.want {
				t.Errorf("SameValueType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddValues(t *testing.T) {
	timeOfDay := func(t string) time.Time {
		tt, err := time.Parse("15:04", t)
		if err != nil {
			panic(err)
		}
		return tt
	}

	tests := []struct {
		name string
		v, d *gen.HealthCheck_Value
		want *gen.HealthCheck_Value
	}{
		{"1+2", IntValue(1), IntValue(2), IntValue(3)},
		{"1-2", IntValue(1), IntValue(-2), IntValue(-1)},
		{"u1+u2", UintValue(1), UintValue(2), UintValue(3)},
		{"1.5+2.5", FloatValue(1.5), FloatValue(2.5), FloatValue(4.0)},
		{"true+false", BoolValue(true), BoolValue(false), BoolValue(true)},
		{"false+true", BoolValue(false), BoolValue(true), BoolValue(false)},
		{`"a"+"b"`, StringValue("a"), StringValue("b"), StringValue("a")},
		{"10:00+2h", TimestampValue(timeOfDay("10:00")), DurationValue(2 * time.Hour), TimestampValue(timeOfDay("12:00"))},
		{"1s+2s", DurationValue(1 * time.Second), DurationValue(2 * time.Second), DurationValue(3 * time.Second)},
		// mismatching types
		{"1+true", IntValue(1), BoolValue(true), IntValue(1)},
		{`1+"a"`, IntValue(1), StringValue("a"), IntValue(1)},
		{`true+"a"`, BoolValue(true), StringValue("a"), BoolValue(true)},
		{"1+2.5", IntValue(1), FloatValue(2.5), IntValue(1)},
		{"1.5+2", FloatValue(1.5), IntValue(2), FloatValue(1.5)},
		{"10:00+11:00", TimestampValue(timeOfDay("10:00")), TimestampValue(timeOfDay("11:00")), TimestampValue(timeOfDay("10:00"))},
		// nil values
		{"nil+1", nil, IntValue(1), nil},
		{"1+nil", IntValue(1), nil, IntValue(1)},
		{"nil+nil", nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddValues(tt.v, tt.d)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("AddValues() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
