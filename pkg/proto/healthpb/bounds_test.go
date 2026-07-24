package healthpb

import (
	"testing"
	"time"
)

func TestValidateValueRange(t *testing.T) {
	tests := []struct {
		name    string
		bounds  *HealthCheck_ValueRange
		wantErr bool
	}{
		// valid cases
		{"low only", &HealthCheck_ValueRange{Low: StringValue("a")}, false},
		{"high only", &HealthCheck_ValueRange{High: UintValue(10)}, false},
		{"low and high", &HealthCheck_ValueRange{Low: IntValue(-5), High: IntValue(10)}, false},
		{"low > high", &HealthCheck_ValueRange{Low: FloatValue(10.5), High: FloatValue(5.5)}, false},
		{"low == high", &HealthCheck_ValueRange{Low: IntValue(5), High: IntValue(5)}, false},
		{"with deadband", &HealthCheck_ValueRange{Low: FloatValue(1.0), High: FloatValue(10.0), Deadband: FloatValue(0.5)}, false},
		{"duration deadband", &HealthCheck_ValueRange{Low: TimestampValue(time.Now()), Deadband: DurationValue(10)}, false},
		// error cases
		{"nil", nil, true},
		{"no bounds", &HealthCheck_ValueRange{}, true},
		{"mismatched types", &HealthCheck_ValueRange{Low: FloatValue(1.0), High: IntValue(10)}, true},
		{"no booleans", &HealthCheck_ValueRange{Low: BoolValue(true)}, true},
		{"db type", &HealthCheck_ValueRange{Low: IntValue(1), Deadband: FloatValue(1.2)}, true},
		{"db timestamp", &HealthCheck_ValueRange{Low: TimestampValue(time.Now()), Deadband: TimestampValue(time.Now())}, true},
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
		closed, high, low       *HealthCheck_ValueRange
		dbClosed, dbHigh, dbLow *HealthCheck_ValueRange
	}{
		closed:   &HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20)},
		high:     &HealthCheck_ValueRange{High: IntValue(20)},
		low:      &HealthCheck_ValueRange{Low: IntValue(10)},
		dbClosed: &HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20), Deadband: IntValue(2)},
		dbHigh:   &HealthCheck_ValueRange{High: IntValue(20), Deadband: IntValue(2)},
		dbLow:    &HealthCheck_ValueRange{Low: IntValue(10), Deadband: IntValue(2)},
	}
	type values struct {
		low, normal, high *HealthCheck_Value
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
		r    *HealthCheck_ValueRange
		v    *HealthCheck_Value
		c    HealthCheck_Normality
		want HealthCheck_Normality
	}{
		// [using this range],[and this current state]->[test a value that should produce this state]
		{"closed,?->normal", r.closed, v.normal, HealthCheck_NORMALITY_UNSPECIFIED, HealthCheck_NORMAL},
		{"closed,?->low", r.closed, v.low, HealthCheck_NORMALITY_UNSPECIFIED, HealthCheck_LOW},
		{"closed,?->high", r.closed, v.high, HealthCheck_NORMALITY_UNSPECIFIED, HealthCheck_HIGH},
		{"closed,normal->normal", r.closed, v.normal, HealthCheck_NORMAL, HealthCheck_NORMAL},
		{"closed,normal->low", r.closed, v.low, HealthCheck_NORMAL, HealthCheck_LOW},
		{"closed,normal->high", r.closed, v.high, HealthCheck_NORMAL, HealthCheck_HIGH},
		{"closed,abnormal->normal", r.closed, v.normal, HealthCheck_ABNORMAL, HealthCheck_NORMAL},
		{"closed,abnormal->low", r.closed, v.low, HealthCheck_ABNORMAL, HealthCheck_LOW},
		{"closed,abnormal->high", r.closed, v.high, HealthCheck_ABNORMAL, HealthCheck_HIGH},
		{"closed,low->normal", r.closed, v.normal, HealthCheck_LOW, HealthCheck_NORMAL},
		{"closed,low->low", r.closed, v.low, HealthCheck_LOW, HealthCheck_LOW},
		{"closed,low->high", r.closed, v.high, HealthCheck_LOW, HealthCheck_HIGH},
		{"closed,high->normal", r.closed, v.normal, HealthCheck_HIGH, HealthCheck_NORMAL},
		{"closed,high->low", r.closed, v.low, HealthCheck_HIGH, HealthCheck_LOW},
		{"closed,high->high", r.closed, v.high, HealthCheck_HIGH, HealthCheck_HIGH},
		{"high,abnormal->normal", r.high, v.normal, HealthCheck_ABNORMAL, HealthCheck_NORMAL},
		{"high,abnormal->high", r.high, v.high, HealthCheck_ABNORMAL, HealthCheck_HIGH},
		{"high,normal->normal", r.high, v.normal, HealthCheck_NORMAL, HealthCheck_NORMAL},
		{"high,normal->high", r.high, v.high, HealthCheck_NORMAL, HealthCheck_HIGH},
		{"high,high->normal", r.high, v.normal, HealthCheck_HIGH, HealthCheck_NORMAL},
		{"high,high->high", r.high, v.high, HealthCheck_HIGH, HealthCheck_HIGH},
		{"low,abnormal->normal", r.low, v.normal, HealthCheck_ABNORMAL, HealthCheck_NORMAL},
		{"low,abnormal->low", r.low, v.low, HealthCheck_ABNORMAL, HealthCheck_LOW},
		{"low,normal->normal", r.low, v.normal, HealthCheck_NORMAL, HealthCheck_NORMAL},
		{"low,normal->low", r.low, v.low, HealthCheck_NORMAL, HealthCheck_LOW},
		{"low,low->normal", r.low, v.normal, HealthCheck_LOW, HealthCheck_NORMAL},
		{"low,low->low", r.low, v.low, HealthCheck_LOW, HealthCheck_LOW},
		{"closed+db,normal->normal", r.dbClosed, v.normal, HealthCheck_NORMAL, HealthCheck_NORMAL},
		{"closed+db,normal->low", r.dbClosed, v.low, HealthCheck_NORMAL, HealthCheck_LOW},
		{"closed+db,normal->high", r.dbClosed, v.high, HealthCheck_NORMAL, HealthCheck_HIGH},
		{"closed+db,low->normal", r.dbClosed, v.dbLow.normal, HealthCheck_LOW, HealthCheck_NORMAL},
		{"closed+db,low->low", r.dbClosed, v.dbLow.low, HealthCheck_LOW, HealthCheck_LOW},
		{"closed+db,low->high", r.dbClosed, v.dbLow.high, HealthCheck_LOW, HealthCheck_HIGH},
		{"closed+db,high->normal", r.dbClosed, v.dbHigh.normal, HealthCheck_HIGH, HealthCheck_NORMAL},
		{"closed+db,high->low", r.dbClosed, v.dbHigh.low, HealthCheck_HIGH, HealthCheck_LOW},
		{"closed+db,high->high", r.dbClosed, v.dbHigh.high, HealthCheck_HIGH, HealthCheck_HIGH},
		{"high+db,normal->normal", r.dbHigh, v.normal, HealthCheck_NORMAL, HealthCheck_NORMAL},
		{"high+db,normal->high", r.dbHigh, v.high, HealthCheck_NORMAL, HealthCheck_HIGH},
		{"high+db,high->normal", r.dbHigh, v.dbHigh.normal, HealthCheck_HIGH, HealthCheck_NORMAL},
		{"high+db,high->high", r.dbHigh, v.dbHigh.high, HealthCheck_HIGH, HealthCheck_HIGH},
		{"low+db,normal->normal", r.dbLow, v.normal, HealthCheck_NORMAL, HealthCheck_NORMAL},
		{"low+db,normal->low", r.dbLow, v.low, HealthCheck_NORMAL, HealthCheck_LOW},
		{"low+db,low->normal", r.dbLow, v.dbLow.normal, HealthCheck_LOW, HealthCheck_NORMAL},
		{"low+db,low->low", r.dbLow, v.dbLow.low, HealthCheck_LOW, HealthCheck_LOW},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := checkRangeState(tt.r, tt.v, tt.c), tt.want; got != want {
				t.Errorf("checkRangeState() = %v, want %v", got, want)
			}
		})
	}
}

func TestNormalRangeCheck_valueToDeviation(t *testing.T) {
	tsBase := time.Unix(1_700_000_000, 0)
	tests := []struct {
		name  string
		r     *HealthCheck_ValueRange
		v     *HealthCheck_Value
		state HealthCheck_Normality
		want  HealthCheck_Deviation
	}{
		// closed range [30,60], width 30 -> ratio = overshoot/30
		{"high tiny -> minor", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(60.2), HealthCheck_HIGH, HealthCheck_MINOR},
		{"high 10% -> moderate", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(63), HealthCheck_HIGH, HealthCheck_MODERATE},
		{"high 25% -> major", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(67.5), HealthCheck_HIGH, HealthCheck_MAJOR},
		{"low 10% -> moderate", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(27), HealthCheck_LOW, HealthCheck_MODERATE},
		{"low 25% -> major", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(22.5), HealthCheck_LOW, HealthCheck_MAJOR},

		// bucket boundaries on [10,20], width 10
		{"just under 10% -> minor", &HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20)}, FloatValue(20.9), HealthCheck_HIGH, HealthCheck_MINOR},
		{"exactly 10% -> moderate", &HealthCheck_ValueRange{Low: FloatValue(10), High: FloatValue(20)}, FloatValue(21), HealthCheck_HIGH, HealthCheck_MODERATE},
		{"just under 25% -> moderate", &HealthCheck_ValueRange{Low: FloatValue(10), High: FloatValue(20)}, FloatValue(22.4), HealthCheck_HIGH, HealthCheck_MODERATE},
		{"exactly 25% -> major", &HealthCheck_ValueRange{Low: FloatValue(10), High: FloatValue(20)}, FloatValue(22.5), HealthCheck_HIGH, HealthCheck_MAJOR},

		// int values
		{"int major", &HealthCheck_ValueRange{Low: IntValue(10), High: IntValue(20)}, IntValue(25), HealthCheck_HIGH, HealthCheck_MAJOR},

		// duration range [10s,20s], width 10s
		{"duration major", &HealthCheck_ValueRange{Low: DurationValue(10 * time.Second), High: DurationValue(20 * time.Second)}, DurationValue(25 * time.Second), HealthCheck_HIGH, HealthCheck_MAJOR},

		// closed timestamp range [t0, t0+10s], width 10s -> 3s over = 30% -> major
		{"timestamp closed major", &HealthCheck_ValueRange{Low: TimestampValue(tsBase), High: TimestampValue(tsBase.Add(10 * time.Second))}, TimestampValue(tsBase.Add(13 * time.Second)), HealthCheck_HIGH, HealthCheck_MAJOR},
		// open-ended timestamp range has no meaningful scale (arbitrary epoch) -> unspecified
		{"timestamp open-ended -> unspecified", &HealthCheck_ValueRange{High: TimestampValue(tsBase)}, TimestampValue(tsBase.Add(time.Hour)), HealthCheck_HIGH, HealthCheck_DEVIATION_UNSPECIFIED},

		// open-ended ranges fall back to |crossed bound|
		{"high-only minor", &HealthCheck_ValueRange{High: FloatValue(20)}, FloatValue(21), HealthCheck_HIGH, HealthCheck_MINOR},
		{"high-only major", &HealthCheck_ValueRange{High: FloatValue(20)}, FloatValue(25), HealthCheck_HIGH, HealthCheck_MAJOR},
		{"low-only moderate", &HealthCheck_ValueRange{Low: FloatValue(100)}, FloatValue(80), HealthCheck_LOW, HealthCheck_MODERATE},

		// zero-width closed range falls back to |bound|
		{"zero width", &HealthCheck_ValueRange{Low: FloatValue(5), High: FloatValue(5)}, FloatValue(6), HealthCheck_HIGH, HealthCheck_MODERATE},

		// state held out of NORMAL by deadband hysteresis while the value is back
		// inside the range -> no excursion -> unspecified
		{"high held by deadband, value in range", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(58), HealthCheck_HIGH, HealthCheck_DEVIATION_UNSPECIFIED},
		{"high state, value at bound", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(60), HealthCheck_HIGH, HealthCheck_DEVIATION_UNSPECIFIED},
		{"low held by deadband, value in range", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(35), HealthCheck_LOW, HealthCheck_DEVIATION_UNSPECIFIED},

		// no meaningful scale / magnitude -> unspecified
		{"zero bound open-ended", &HealthCheck_ValueRange{High: FloatValue(0)}, FloatValue(5), HealthCheck_HIGH, HealthCheck_DEVIATION_UNSPECIFIED},
		{"string bounds", &HealthCheck_ValueRange{Low: StringValue("a"), High: StringValue("z")}, StringValue("~"), HealthCheck_HIGH, HealthCheck_DEVIATION_UNSPECIFIED},
		{"normal -> unspecified", &HealthCheck_ValueRange{Low: FloatValue(30), High: FloatValue(60)}, FloatValue(45), HealthCheck_NORMAL, HealthCheck_DEVIATION_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &normalRangeCheck{bounds: tt.r}
			if got := c.valueToDeviation(tt.v, tt.state); got != tt.want {
				t.Errorf("valueToDeviation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueCheck_valueToDeviation(t *testing.T) {
	// equality and value-set checks have no magnitude -> always unspecified
	vc := &valueCheck{value: IntValue(1), eq: HealthCheck_NORMAL, neq: HealthCheck_ABNORMAL}
	if got := vc.valueToDeviation(IntValue(99), HealthCheck_ABNORMAL); got != HealthCheck_DEVIATION_UNSPECIFIED {
		t.Errorf("valueCheck.valueToDeviation() = %v, want DEVIATION_UNSPECIFIED", got)
	}
	vsc := &valuesCheck{values: []*HealthCheck_Value{IntValue(1)}, in: HealthCheck_NORMAL, nin: HealthCheck_ABNORMAL}
	if got := vsc.valueToDeviation(IntValue(99), HealthCheck_ABNORMAL); got != HealthCheck_DEVIATION_UNSPECIFIED {
		t.Errorf("valuesCheck.valueToDeviation() = %v, want DEVIATION_UNSPECIFIED", got)
	}
}

func Test_less(t *testing.T) {
	tests := []struct {
		name string
		x, y *HealthCheck_Value
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
