// Package scale builds the sim driver's configurable working-hours day curve.
//
// Where the mock driver uses a single hard-coded NineToFive curve, WorkingHours
// builds a scaler for configurable working hours and days. Both drivers share the
// underlying mechanism (the Time/TimeFunc list and the ramp helpers) from
// pkg/util/timescale.
package scale

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/timescale"
)

// Time aliases the shared timescale type so sim code can refer to scale.Time.
type Time = timescale.Time

const (
	// offPeak is the baseline factor reported outside working hours/days.
	// Zero means the building is empty overnight and at weekends; base electrical
	// load still keeps energy use non-zero.
	offPeak = 0.0
	// peak is the factor reported during core working hours.
	peak = 1.0
	// lunchLow is the factor reached at the bottom of the midday lunch dip.
	lunchLow = 0.6
)

// WorkingHours builds a Time scaler that reports higher values during the
// configured working hours on working days. start and end are hours of the day
// in 24h decimal form (e.g. 8.5 == 08:30). days lists the working weekdays
// (empty means Monday–Friday); outside those days the scaler reports offPeak.
//
// The curve ramps from offPeak up to peak over the first part of the day, dips
// to lunchLow around the midpoint, then ramps back down to offPeak by end.
func WorkingHours(start, end float64, days []time.Weekday) Time {
	// A degenerate window is treated as "always peak on working days".
	if end <= start {
		return Time{timescale.WorkdayRamp(offPeak, peak, days)}
	}

	rampDur := min((end-start)/4, 2)
	mid := (start + end) / 2
	lunchHalf := 0.5
	if lunchHalf > rampDur {
		lunchHalf = rampDur / 2
	}

	return Time{
		timescale.WorkdayRamp(offPeak, peak, days),                    // non-working day => offPeak, must be first
		timescale.LinearRampHour(start, start+rampDur, offPeak, peak), // morning ramp up
		timescale.LinearRampHour(mid-lunchHalf, mid, peak, lunchLow),  // into lunch
		timescale.LinearRampHour(mid, mid+lunchHalf, lunchLow, peak),  // out of lunch
		timescale.LinearRampHour(end-rampDur, end, peak, offPeak),     // evening ramp down
	}
}
