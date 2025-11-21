package healthpb

import (
	"testing"
	"time"

	"github.com/vanti-dev/sc-bos/pkg/gen"
)

func TestValidateValueRange(t *testing.T) {
	tests := []struct {
		name    string
		bounds  *gen.HealthCheck_ValueRange
		wantErr bool
	}{
		// valid cases
		{"low only", &gen.HealthCheck_ValueRange{Low: StringValue("a")}, false},
		{"high only", &gen.HealthCheck_ValueRange{High: UintValue(10)}, false},
		{"low and high", &gen.HealthCheck_ValueRange{Low: IntValue(-5), High: IntValue(10)}, false},
		{"low > high", &gen.HealthCheck_ValueRange{Low: FloatValue(10.5), High: FloatValue(5.5)}, false},
		{"low == high", &gen.HealthCheck_ValueRange{Low: IntValue(5), High: IntValue(5)}, false},
		{"with deadband", &gen.HealthCheck_ValueRange{Low: FloatValue(1.0), High: FloatValue(10.0), Deadband: FloatValue(0.5)}, false},
		{"duration deadband", &gen.HealthCheck_ValueRange{Low: TimestampValue(time.Now()), Deadband: DurationValue(10)}, false},
		// error cases
		{"nil", nil, true},
		{"no bounds", &gen.HealthCheck_ValueRange{}, true},
		{"mismatched types", &gen.HealthCheck_ValueRange{Low: FloatValue(1.0), High: IntValue(10)}, true},
		{"no booleans", &gen.HealthCheck_ValueRange{Low: BoolValue(true)}, true},
		{"db type", &gen.HealthCheck_ValueRange{Low: IntValue(1), Deadband: FloatValue(1.2)}, true},
		{"db timestamp", &gen.HealthCheck_ValueRange{Low: TimestampValue(time.Now()), Deadband: TimestampValue(time.Now())}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValueRange(tt.bounds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValueRange() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckRangeState(t *testing.T) {
	r := struct {
		closed, high, low       *gen.HealthCheck_ValueRange
		dbClosed, dbHigh, dbLow *gen.HealthCheck_ValueRange
	}{
		closed:   &gen.HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20)},
		high:     &gen.HealthCheck_ValueRange{High: IntValue(20)},
		low:      &gen.HealthCheck_ValueRange{Low: IntValue(10)},
		dbClosed: &gen.HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20), Deadband: IntValue(2)},
		dbHigh:   &gen.HealthCheck_ValueRange{High: IntValue(20), Deadband: IntValue(2)},
		dbLow:    &gen.HealthCheck_ValueRange{Low: IntValue(10), Deadband: IntValue(2)},
	}
	type values struct {
		low, normal, high *gen.HealthCheck_Value
	}
	v := struct {
		values
		dbLow, dbHigh values
	}{
		values: values{
			low:    IntValue(9),
			normal: IntValue(15),
			high:   IntValue(21),
		},
		// values for when there is a deadband and the current state is low
		dbLow: values{
			low:    IntValue(11),
			normal: IntValue(12),
			high:   IntValue(21),
		},
		// values for when there is a deadband and the current state is high
		dbHigh: values{
			low:    IntValue(9),
			normal: IntValue(18),
			high:   IntValue(19),
		},
	}

	tests := []struct {
		name string
		r    *gen.HealthCheck_ValueRange
		v    *gen.HealthCheck_Value
		c    gen.HealthCheck_Normality
		want gen.HealthCheck_Normality
	}{
		// [using this range],[and this current state]->[test a value that should produce this state]
		{"closed,?->normal", r.closed, v.normal, gen.HealthCheck_NORMALITY_UNSPECIFIED, gen.HealthCheck_NORMAL},
		{"closed,?->low", r.closed, v.low, gen.HealthCheck_NORMALITY_UNSPECIFIED, gen.HealthCheck_LOW},
		{"closed,?->high", r.closed, v.high, gen.HealthCheck_NORMALITY_UNSPECIFIED, gen.HealthCheck_HIGH},
		{"closed,normal->normal", r.closed, v.normal, gen.HealthCheck_NORMAL, gen.HealthCheck_NORMAL},
		{"closed,normal->low", r.closed, v.low, gen.HealthCheck_NORMAL, gen.HealthCheck_LOW},
		{"closed,normal->high", r.closed, v.high, gen.HealthCheck_NORMAL, gen.HealthCheck_HIGH},
		{"closed,abnormal->normal", r.closed, v.normal, gen.HealthCheck_ABNORMAL, gen.HealthCheck_NORMAL},
		{"closed,abnormal->low", r.closed, v.low, gen.HealthCheck_ABNORMAL, gen.HealthCheck_LOW},
		{"closed,abnormal->high", r.closed, v.high, gen.HealthCheck_ABNORMAL, gen.HealthCheck_HIGH},
		{"closed,low->normal", r.closed, v.normal, gen.HealthCheck_LOW, gen.HealthCheck_NORMAL},
		{"closed,low->low", r.closed, v.low, gen.HealthCheck_LOW, gen.HealthCheck_LOW},
		{"closed,low->high", r.closed, v.high, gen.HealthCheck_LOW, gen.HealthCheck_HIGH},
		{"closed,high->normal", r.closed, v.normal, gen.HealthCheck_HIGH, gen.HealthCheck_NORMAL},
		{"closed,high->low", r.closed, v.low, gen.HealthCheck_HIGH, gen.HealthCheck_LOW},
		{"closed,high->high", r.closed, v.high, gen.HealthCheck_HIGH, gen.HealthCheck_HIGH},
		{"high,abnormal->normal", r.high, v.normal, gen.HealthCheck_ABNORMAL, gen.HealthCheck_NORMAL},
		{"high,abnormal->high", r.high, v.high, gen.HealthCheck_ABNORMAL, gen.HealthCheck_HIGH},
		{"high,normal->normal", r.high, v.normal, gen.HealthCheck_NORMAL, gen.HealthCheck_NORMAL},
		{"high,normal->high", r.high, v.high, gen.HealthCheck_NORMAL, gen.HealthCheck_HIGH},
		{"high,high->normal", r.high, v.normal, gen.HealthCheck_HIGH, gen.HealthCheck_NORMAL},
		{"high,high->high", r.high, v.high, gen.HealthCheck_HIGH, gen.HealthCheck_HIGH},
		{"low,abnormal->normal", r.low, v.normal, gen.HealthCheck_ABNORMAL, gen.HealthCheck_NORMAL},
		{"low,abnormal->low", r.low, v.low, gen.HealthCheck_ABNORMAL, gen.HealthCheck_LOW},
		{"low,normal->normal", r.low, v.normal, gen.HealthCheck_NORMAL, gen.HealthCheck_NORMAL},
		{"low,normal->low", r.low, v.low, gen.HealthCheck_NORMAL, gen.HealthCheck_LOW},
		{"low,low->normal", r.low, v.normal, gen.HealthCheck_LOW, gen.HealthCheck_NORMAL},
		{"low,low->low", r.low, v.low, gen.HealthCheck_LOW, gen.HealthCheck_LOW},
		{"closed+db,normal->normal", r.dbClosed, v.normal, gen.HealthCheck_NORMAL, gen.HealthCheck_NORMAL},
		{"closed+db,normal->low", r.dbClosed, v.low, gen.HealthCheck_NORMAL, gen.HealthCheck_LOW},
		{"closed+db,normal->high", r.dbClosed, v.high, gen.HealthCheck_NORMAL, gen.HealthCheck_HIGH},
		{"closed+db,low->normal", r.dbClosed, v.dbLow.normal, gen.HealthCheck_LOW, gen.HealthCheck_NORMAL},
		{"closed+db,low->low", r.dbClosed, v.dbLow.low, gen.HealthCheck_LOW, gen.HealthCheck_LOW},
		{"closed+db,low->high", r.dbClosed, v.dbLow.high, gen.HealthCheck_LOW, gen.HealthCheck_HIGH},
		{"closed+db,high->normal", r.dbClosed, v.dbHigh.normal, gen.HealthCheck_HIGH, gen.HealthCheck_NORMAL},
		{"closed+db,high->low", r.dbClosed, v.dbHigh.low, gen.HealthCheck_HIGH, gen.HealthCheck_LOW},
		{"closed+db,high->high", r.dbClosed, v.dbHigh.high, gen.HealthCheck_HIGH, gen.HealthCheck_HIGH},
		{"high+db,normal->normal", r.dbHigh, v.normal, gen.HealthCheck_NORMAL, gen.HealthCheck_NORMAL},
		{"high+db,normal->high", r.dbHigh, v.high, gen.HealthCheck_NORMAL, gen.HealthCheck_HIGH},
		{"high+db,high->normal", r.dbHigh, v.dbHigh.normal, gen.HealthCheck_HIGH, gen.HealthCheck_NORMAL},
		{"high+db,high->high", r.dbHigh, v.dbHigh.high, gen.HealthCheck_HIGH, gen.HealthCheck_HIGH},
		{"low+db,normal->normal", r.dbLow, v.normal, gen.HealthCheck_NORMAL, gen.HealthCheck_NORMAL},
		{"low+db,normal->low", r.dbLow, v.low, gen.HealthCheck_NORMAL, gen.HealthCheck_LOW},
		{"low+db,low->normal", r.dbLow, v.dbLow.normal, gen.HealthCheck_LOW, gen.HealthCheck_NORMAL},
		{"low+db,low->low", r.dbLow, v.dbLow.low, gen.HealthCheck_LOW, gen.HealthCheck_LOW},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := checkRangeState(tt.r, tt.v, tt.c), tt.want; got != want {
				t.Errorf("checkRangeState() = %v, want %v", got, want)
			}
		})
	}
}

func Test_less(t *testing.T) {
	tests := []struct {
		name string
		x, y *gen.HealthCheck_Value
		want bool
	}{
		{"int<int", IntValue(5), IntValue(6), true},
		{"int==int", IntValue(5), IntValue(5), false},
		{"int>int", IntValue(6), IntValue(5), false},
		{"uint<uint", UintValue(5), UintValue(6), true},
		{"uint==uint", UintValue(5), UintValue(5), false},
		{"uint>uint", UintValue(6), UintValue(5), false},
		{"float<float", FloatValue(5.5), FloatValue(6.5), true},
		{"float==float", FloatValue(5.5), FloatValue(5.5), false},
		{"float>float", FloatValue(6.5), FloatValue(5.5), false},
		{"string<string", StringValue("a"), StringValue("b"), true},
		{"string==string", StringValue("a"), StringValue("a"), false},
		{"string>string", StringValue("b"), StringValue("a"), false},
		{"time<time", TimestampValue(time.Unix(1000, 0)), TimestampValue(time.Unix(2000, 0)), true},
		{"time==time", TimestampValue(time.Unix(1000, 0)), TimestampValue(time.Unix(1000, 0)), false},
		{"time>time", TimestampValue(time.Unix(2000, 0)), TimestampValue(time.Unix(1000, 0)), false},
		{"duration<duration", DurationValue(5), DurationValue(10), true},
		{"duration==duration", DurationValue(5), DurationValue(5), false},
		{"duration>duration", DurationValue(10), DurationValue(5), false},
		{"bool", BoolValue(true), BoolValue(false), false},
		{"bool equal", BoolValue(true), BoolValue(true), false},
		{"mixed types", IntValue(5), FloatValue(5.0), false},
		{"nil x", nil, IntValue(5), false},
		{"nil y", IntValue(5), nil, false},
		{"both nil", nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := less(tt.x, tt.y); got != tt.want {
				t.Errorf("less() = %v, want %v", got, tt.want)
			}
		})
	}
}
