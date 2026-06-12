package timescale

import (
	"testing"
	"time"
)

var (
	monday   = time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC) // a Monday
	saturday = time.Date(2024, 1, 13, 0, 0, 0, 0, time.UTC)
)

func at(day time.Time, hour int) time.Time {
	return day.Add(time.Duration(hour) * time.Hour)
}

func TestAt_ReturnsFirstNonPositiveTense(t *testing.T) {
	// A func with tense -1 (before its window) should short-circuit; a func with
	// tense 1 (after) should defer to the next.
	s := Time{
		func(time.Time) (float64, int) { return 0.9, 1 },  // defer
		func(time.Time) (float64, int) { return 0.3, -1 }, // win
		func(time.Time) (float64, int) { return 0.7, 0 },  // not reached
	}
	if v := s.At(monday); v != 0.3 {
		t.Errorf("At = %g, want 0.3 (first non-positive tense)", v)
	}
}

func TestLinearRampHour(t *testing.T) {
	f := LinearRampHour(8, 10, 0.0, 1.0)
	cases := []struct {
		hour      int
		wantScale float64
		wantTense int
	}{
		{6, 0.0, -1}, // before the window
		{9, 0.5, 0},  // midway
		{12, 1.0, 1}, // after the window
	}
	for _, c := range cases {
		scale, tense := f(at(monday, c.hour))
		if scale != c.wantScale || tense != c.wantTense {
			t.Errorf("%02d:00 = (%g, %d), want (%g, %d)", c.hour, scale, tense, c.wantScale, c.wantTense)
		}
	}
}

func TestWorkdayRamp_DefaultsToWeekdays(t *testing.T) {
	f := WorkdayRamp(0.1, 1, nil) // nil => Mon–Fri (the behaviour mock's NineToFive relies on)
	if scale, tense := f(saturday); scale != 0.1 || tense != 0 {
		t.Errorf("Saturday = (%g, %d), want (0.1, 0)", scale, tense)
	}
	if scale, tense := f(monday); scale != 1 || tense != 1 {
		t.Errorf("Monday = (%g, %d), want (1, 1)", scale, tense)
	}
}

func TestWorkdayRamp_CustomDays(t *testing.T) {
	f := WorkdayRamp(0, 1, []time.Weekday{time.Saturday})
	if scale, _ := f(saturday); scale != 1 {
		t.Errorf("Saturday as a working day = %g, want 1", scale)
	}
	if scale, tense := f(monday); scale != 0 || tense != 0 {
		t.Errorf("Monday (not configured) = (%g, %d), want (0, 0)", scale, tense)
	}
}
