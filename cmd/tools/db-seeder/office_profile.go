package main

import (
	"math"
	"math/rand"
	"time"
)

// officeLoad returns a factor in [0.0, 1.0] representing how busy the office is at time t.
// 0.0 = empty building, 1.0 = full peak occupancy.
// The return value is a smooth deterministic envelope; callers add their own noise.
func officeLoad(t time.Time) float64 {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return 0.05
	}
	hour := float64(t.Hour()) + float64(t.Minute())/60.0
	switch {
	case hour < 7.0:
		return 0.0
	case hour < 9.0:
		return lerp(hour, 7, 9, 0.0, 0.9)
	case hour < 12.0:
		return 0.9
	case hour < 13.0:
		return lerp(hour, 12, 13, 0.9, 0.45)
	case hour < 14.0:
		return lerp(hour, 13, 14, 0.45, 0.9)
	case hour < 17.0:
		return 0.9
	case hour < 19.0:
		return lerp(hour, 17, 19, 0.9, 0.0)
	default:
		return 0.0
	}
}

// lerp maps v from [fromLow, fromHigh] to [toLow, toHigh].
func lerp(v, fromLow, fromHigh, toLow, toHigh float64) float64 {
	return (v-fromLow)/(fromHigh-fromLow)*(toHigh-toLow) + toLow
}

// intervalForLoad returns a sampling interval that is shorter when the office is active.
func intervalForLoad(load float64) time.Duration {
	if load < 0.1 {
		return time.Duration(30+rand.Intn(31)) * time.Minute
	}
	if load < 0.5 {
		return time.Duration(10+rand.Intn(21)) * time.Minute
	}
	return time.Duration(5+rand.Intn(16)) * time.Minute
}

// clampFloat64 clamps v to [lo, hi].
func clampFloat64(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
