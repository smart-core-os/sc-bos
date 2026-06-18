// Package timescale maps a time-of-day to a [0,1] activity/occupancy factor.
//
// It is the shared mechanism behind the mock and sim drivers' day curves: a Time
// is an ordered list of TimeFuncs, and At returns the value of the first func
// whose tense is non-positive. The drivers build their own curves on top of the
// LinearRampHour and WorkdayRamp helpers.
package timescale

import (
	"time"
)

// TimeFunc returns a scale factor based on a time.
// The scale return value will be in the range [0,1].
// If t is before the period represented by this scale func, tense will be -1, 0 when during, and 1 when t is after.
// A tense of 0 should be returned if the func applies to the entire timeline.
type TimeFunc func(t time.Time) (scale float64, tense int)

// Time scales based on time.Time values.
// The first entry that returns a non-positive tense return value will have its value returned.
type Time []TimeFunc

// Now returns a scale factor between [0, 1] for the current time.
func (s Time) Now() float64 {
	return s.At(time.Now())
}

// At returns a scale factor between [0, 1] for the given time.
func (s Time) At(t time.Time) float64 {
	var v float64
	for _, timeFunc := range s {
		var cmp int
		v, cmp = timeFunc(t)
		if cmp <= 0 {
			return v
		}
	}
	return v
}

// LinearRampHour returns a TimeFunc that ramps linearly from lo to hi between
// the from and to hours of the day (24h decimal form, e.g. 8.5 == 08:30). Before
// from it reports lo with tense -1; after to it reports hi with tense 1.
func LinearRampHour(from, to, lo, hi float64) TimeFunc {
	return func(t time.Time) (float64, int) {
		hour := float64(t.Hour()) + float64(t.Minute())/60.0
		if hour < from {
			return lo, -1
		}
		if hour > to {
			return hi, 1
		}
		return mapRange(hour, from, to, lo, hi), 0
	}
}

// WorkdayRamp reports lo on non-working days (tense 0, terminal) and hi on
// working days (tense 1, deferring to the hour-based ramps that follow it). days
// lists the working weekdays; an empty list means Monday–Friday.
func WorkdayRamp(lo, hi float64, days []time.Weekday) TimeFunc {
	if len(days) == 0 {
		days = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	}
	working := make(map[time.Weekday]bool, len(days))
	for _, d := range days {
		working[d] = true
	}
	return func(t time.Time) (float64, int) {
		if !working[t.Weekday()] {
			return lo, 0
		}
		return hi, 1
	}
}

// mapRange maps a value from one range to another.
func mapRange(value, fromLow, fromHigh, toLow, toHigh float64) float64 {
	return (value-fromLow)/(fromHigh-fromLow)*(toHigh-toLow) + toLow
}
