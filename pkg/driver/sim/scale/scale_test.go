package scale

import (
	"testing"
	"time"
)

func TestWorkingHours(t *testing.T) {
	mon := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC) // Monday
	sat := time.Date(2024, 1, 13, 0, 0, 0, 0, time.UTC)
	s := WorkingHours(8, 18, nil)

	at := func(d time.Time, h int) time.Time { return d.Add(time.Duration(h) * time.Hour) }

	if v := s.At(at(sat, 13)); v != offPeak {
		t.Errorf("weekend midday = %g, want offPeak %g", v, offPeak)
	}
	if v := s.At(at(mon, 3)); v != offPeak {
		t.Errorf("weekday 03:00 = %g, want offPeak %g", v, offPeak)
	}
	// 11:00 is past the morning ramp and before the midday lunch dip → peak.
	peakVal := s.At(at(mon, 11))
	if peakVal < 0.95 {
		t.Errorf("weekday 11:00 = %g, want near peak", peakVal)
	}
	if morning := s.At(at(mon, 9)); morning <= offPeak || morning >= peakVal {
		t.Errorf("weekday 09:00 = %g, want ramping between offPeak and peak %g", morning, peakVal)
	}
	// Lunch dip: the midpoint (13:00 for an 08:00–18:00 day) is the bottom of the dip.
	if lunch := s.At(at(mon, 13)); lunch >= peakVal {
		t.Errorf("lunch dip 13:00 = %g, want below peak %g", lunch, peakVal)
	}
}

func TestWorkingHours_CustomDays(t *testing.T) {
	sun := time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC) // Sunday
	s := WorkingHours(9, 17, []time.Weekday{time.Sunday})
	if v := s.At(sun); v < 0.95 {
		t.Errorf("Sunday midday with Sunday as a working day = %g, want near peak", v)
	}
}
