package healthpb

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func TestValidateValueRange(t *testing.T) {
	tests := []struct {
		name    string
		bounds  *healthpb.HealthCheck_ValueRange
		wantErr bool
	}{
		// valid cases
		{"low only", &healthpb.HealthCheck_ValueRange{Low: StringValue("a")}, false},
		{"high only", &healthpb.HealthCheck_ValueRange{High: UintValue(10)}, false},
		{"low and high", &healthpb.HealthCheck_ValueRange{Low: IntValue(-5), High: IntValue(10)}, false},
		{"low > high", &healthpb.HealthCheck_ValueRange{Low: FloatValue(10.5), High: FloatValue(5.5)}, false},
		{"low == high", &healthpb.HealthCheck_ValueRange{Low: IntValue(5), High: IntValue(5)}, false},
		{"with deadband", &healthpb.HealthCheck_ValueRange{Low: FloatValue(1.0), High: FloatValue(10.0), Deadband: FloatValue(0.5)}, false},
		{"duration deadband", &healthpb.HealthCheck_ValueRange{Low: TimestampValue(time.Now()), Deadband: DurationValue(10)}, false},
		// error cases
		{"nil", nil, true},
		{"no bounds", &healthpb.HealthCheck_ValueRange{}, true},
		{"mismatched types", &healthpb.HealthCheck_ValueRange{Low: FloatValue(1.0), High: IntValue(10)}, true},
		{"no booleans", &healthpb.HealthCheck_ValueRange{Low: BoolValue(true)}, true},
		{"db type", &healthpb.HealthCheck_ValueRange{Low: IntValue(1), Deadband: FloatValue(1.2)}, true},
		{"db timestamp", &healthpb.HealthCheck_ValueRange{Low: TimestampValue(time.Now()), Deadband: TimestampValue(time.Now())}, true},
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
		closed, high, low       *healthpb.HealthCheck_ValueRange
		dbClosed, dbHigh, dbLow *healthpb.HealthCheck_ValueRange
	}{
		closed:   &healthpb.HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20)},
		high:     &healthpb.HealthCheck_ValueRange{High: IntValue(20)},
		low:      &healthpb.HealthCheck_ValueRange{Low: IntValue(10)},
		dbClosed: &healthpb.HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20), Deadband: IntValue(2)},
		dbHigh:   &healthpb.HealthCheck_ValueRange{High: IntValue(20), Deadband: IntValue(2)},
		dbLow:    &healthpb.HealthCheck_ValueRange{Low: IntValue(10), Deadband: IntValue(2)},
	}
	type values struct {
		low, normal, high *healthpb.HealthCheck_Value
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
		r    *healthpb.HealthCheck_ValueRange
		v    *healthpb.HealthCheck_Value
		c    healthpb.HealthCheck_Normality
		want healthpb.HealthCheck_Normality
	}{
		// [using this range],[and this current state]->[test a value that should produce this state]
		{"closed,?->normal", r.closed, v.normal, healthpb.HealthCheck_NORMALITY_UNSPECIFIED, healthpb.HealthCheck_NORMAL},
		{"closed,?->low", r.closed, v.low, healthpb.HealthCheck_NORMALITY_UNSPECIFIED, healthpb.HealthCheck_LOW},
		{"closed,?->high", r.closed, v.high, healthpb.HealthCheck_NORMALITY_UNSPECIFIED, healthpb.HealthCheck_HIGH},
		{"closed,normal->normal", r.closed, v.normal, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_NORMAL},
		{"closed,normal->low", r.closed, v.low, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_LOW},
		{"closed,normal->high", r.closed, v.high, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_HIGH},
		{"closed,abnormal->normal", r.closed, v.normal, healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_NORMAL},
		{"closed,abnormal->low", r.closed, v.low, healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_LOW},
		{"closed,abnormal->high", r.closed, v.high, healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_HIGH},
		{"closed,low->normal", r.closed, v.normal, healthpb.HealthCheck_LOW, healthpb.HealthCheck_NORMAL},
		{"closed,low->low", r.closed, v.low, healthpb.HealthCheck_LOW, healthpb.HealthCheck_LOW},
		{"closed,low->high", r.closed, v.high, healthpb.HealthCheck_LOW, healthpb.HealthCheck_HIGH},
		{"closed,high->normal", r.closed, v.normal, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_NORMAL},
		{"closed,high->low", r.closed, v.low, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_LOW},
		{"closed,high->high", r.closed, v.high, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_HIGH},
		{"high,abnormal->normal", r.high, v.normal, healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_NORMAL},
		{"high,abnormal->high", r.high, v.high, healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_HIGH},
		{"high,normal->normal", r.high, v.normal, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_NORMAL},
		{"high,normal->high", r.high, v.high, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_HIGH},
		{"high,high->normal", r.high, v.normal, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_NORMAL},
		{"high,high->high", r.high, v.high, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_HIGH},
		{"low,abnormal->normal", r.low, v.normal, healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_NORMAL},
		{"low,abnormal->low", r.low, v.low, healthpb.HealthCheck_ABNORMAL, healthpb.HealthCheck_LOW},
		{"low,normal->normal", r.low, v.normal, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_NORMAL},
		{"low,normal->low", r.low, v.low, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_LOW},
		{"low,low->normal", r.low, v.normal, healthpb.HealthCheck_LOW, healthpb.HealthCheck_NORMAL},
		{"low,low->low", r.low, v.low, healthpb.HealthCheck_LOW, healthpb.HealthCheck_LOW},
		{"closed+db,normal->normal", r.dbClosed, v.normal, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_NORMAL},
		{"closed+db,normal->low", r.dbClosed, v.low, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_LOW},
		{"closed+db,normal->high", r.dbClosed, v.high, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_HIGH},
		{"closed+db,low->normal", r.dbClosed, v.dbLow.normal, healthpb.HealthCheck_LOW, healthpb.HealthCheck_NORMAL},
		{"closed+db,low->low", r.dbClosed, v.dbLow.low, healthpb.HealthCheck_LOW, healthpb.HealthCheck_LOW},
		{"closed+db,low->high", r.dbClosed, v.dbLow.high, healthpb.HealthCheck_LOW, healthpb.HealthCheck_HIGH},
		{"closed+db,high->normal", r.dbClosed, v.dbHigh.normal, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_NORMAL},
		{"closed+db,high->low", r.dbClosed, v.dbHigh.low, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_LOW},
		{"closed+db,high->high", r.dbClosed, v.dbHigh.high, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_HIGH},
		{"high+db,normal->normal", r.dbHigh, v.normal, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_NORMAL},
		{"high+db,normal->high", r.dbHigh, v.high, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_HIGH},
		{"high+db,high->normal", r.dbHigh, v.dbHigh.normal, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_NORMAL},
		{"high+db,high->high", r.dbHigh, v.dbHigh.high, healthpb.HealthCheck_HIGH, healthpb.HealthCheck_HIGH},
		{"low+db,normal->normal", r.dbLow, v.normal, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_NORMAL},
		{"low+db,normal->low", r.dbLow, v.low, healthpb.HealthCheck_NORMAL, healthpb.HealthCheck_LOW},
		{"low+db,low->normal", r.dbLow, v.dbLow.normal, healthpb.HealthCheck_LOW, healthpb.HealthCheck_NORMAL},
		{"low+db,low->low", r.dbLow, v.dbLow.low, healthpb.HealthCheck_LOW, healthpb.HealthCheck_LOW},
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
		x, y *healthpb.HealthCheck_Value
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
