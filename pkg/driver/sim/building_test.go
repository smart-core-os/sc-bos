package sim

import (
	"math"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/sim/scale"
)

// A weekday and a weekend day for time-of-day tests (Jan 8 2024 is a Monday).
var (
	monday   = time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
	saturday = time.Date(2024, 1, 13, 0, 0, 0, 0, time.UTC)
)

func at(day time.Time, hour, min int) time.Time {
	return day.Add(time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute)
}

const (
	testBaseLoadW      = 5000
	testRatedLightingW = 600
	testRatedFcuW      = 800
)

// newTestBuilding builds a single-room building with known rated capacities so
// tests can assert the energy coupling exactly.
func newTestBuilding(start time.Time) *Building {
	room := &Room{
		Name:           "open-plan",
		MaxOccupancy:   100,
		SetPointC:      21,
		ratedLightingW: testRatedLightingW,
		ratedFcuW:      testRatedFcuW,
	}
	floor := &Floor{Name: "ground", Rooms: []*Room{room}}
	scaler := scale.WorkingHours(8, 18, nil)
	return NewBuilding(start, scaler, testBaseLoadW, 1, []*Floor{floor})
}

func (b *Building) room() *Room { return b.Floors[0].Rooms[0] }

func TestNewBuilding_TimeOfDay(t *testing.T) {
	tests := []struct {
		name        string
		start       time.Time
		wantPeople  string // "zero", "some", "many"
		wantDemandW string // "base" or "above"
	}{
		{"overnight", at(monday, 3, 0), "zero", "base"},
		{"midday", at(monday, 13, 0), "many", "above"},
		{"weekend", at(saturday, 13, 0), "zero", "base"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestBuilding(tt.start)
			r := b.room()
			switch tt.wantPeople {
			case "zero":
				if r.Occupants != 0 {
					t.Errorf("Occupants = %d, want 0", r.Occupants)
				}
			case "many":
				if r.Occupants < 50 {
					t.Errorf("Occupants = %d, want many (>=50)", r.Occupants)
				}
			}
			switch tt.wantDemandW {
			case "base":
				if b.DemandW != testBaseLoadW {
					t.Errorf("DemandW = %g, want base %d", b.DemandW, testBaseLoadW)
				}
				if r.LightLevel != 0 {
					t.Errorf("LightLevel = %g, want 0 when empty/off-hours", r.LightLevel)
				}
			case "above":
				if b.DemandW <= testBaseLoadW {
					t.Errorf("DemandW = %g, want above base %d", b.DemandW, testBaseLoadW)
				}
				if r.LightLevel <= 0 {
					t.Errorf("LightLevel = %g, want > 0 when occupied", r.LightLevel)
				}
			}
		})
	}
}

func TestTick_DemandIdentity(t *testing.T) {
	b := newTestBuilding(at(monday, 8, 0))
	now := at(monday, 8, 0)
	for i := 0; i < 60; i++ {
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)

		// Derive the expected demand independently from the physical state (light
		// level, fan speed and the known rated capacities) rather than reading the
		// engine's own cached lightsW/fcuW, so a wrong load derivation is caught.
		r := b.room()
		want := testBaseLoadW + r.LightLevel/100*testRatedLightingW + r.FanPct/100*testRatedFcuW
		if math.Abs(b.DemandW-want) > 1e-9 {
			t.Fatalf("tick %d: DemandW = %g, want base + derived loads = %g (light=%g%% fan=%g%%)",
				i, b.DemandW, want, r.LightLevel, r.FanPct)
		}
	}
}

func TestTick_MeterMonotonic(t *testing.T) {
	b := newTestBuilding(at(monday, 9, 0))
	now := at(monday, 9, 0)
	prev := b.MeterKWh
	for i := 0; i < 240; i++ {
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
		if b.MeterKWh < prev {
			t.Fatalf("tick %d: MeterKWh went backwards: %g < %g", i, b.MeterKWh, prev)
		}
		prev = b.MeterKWh
	}
	// Over 4 hours with base load alone the meter must have accumulated at least
	// baseLoad(kW) * 4h = 5kW * 4h = 20 kWh.
	if b.MeterKWh < 20 {
		t.Errorf("MeterKWh = %g, want >= 20 after 4h", b.MeterKWh)
	}
}

func TestTick_OccupancyDrivesEnergy(t *testing.T) {
	// Fill the building up over the morning, then empty it overnight and confirm
	// demand collapses back to the base load and lights switch off.
	b := newTestBuilding(at(monday, 7, 0))
	now := at(monday, 7, 0)

	// Run through to midday: occupancy and demand should climb.
	for now.Before(at(monday, 12, 0)) {
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
	}
	if b.room().Occupants < 50 {
		t.Fatalf("midday Occupants = %d, want busy", b.room().Occupants)
	}
	if b.DemandW <= testBaseLoadW {
		t.Fatalf("midday DemandW = %g, want above base", b.DemandW)
	}

	// Jump to the early hours and let everyone leave.
	now = at(monday, 2, 0).Add(24 * time.Hour) // next day 02:00
	for i := 0; i < 120; i++ {
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
	}
	if b.room().Occupants != 0 {
		t.Errorf("overnight Occupants = %d, want 0", b.room().Occupants)
	}
	if b.room().LightLevel != 0 {
		t.Errorf("overnight LightLevel = %g, want 0", b.room().LightLevel)
	}
	if math.Abs(b.DemandW-testBaseLoadW) > 1e-6 {
		t.Errorf("overnight DemandW = %g, want base %d", b.DemandW, testBaseLoadW)
	}
}

func TestTick_SteadyOccupancyNoChurn(t *testing.T) {
	// A room at steady occupancy must not manufacture enter/leave events or flicker
	// its people count every tick. Observe a flat peak window (14:30–15:45 for an
	// 08:00–18:00 day: past lunch recovery, before the 16:00 evening ramp).
	b := newTestBuilding(at(monday, 14, 0))
	now := at(monday, 14, 0)
	for i := 0; i < 30; i++ { // settle to 14:30
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
	}
	r := b.room()
	enter0, leave0, occMin := r.EnterTotal, r.LeaveTotal, r.Occupants
	for i := 0; i < 75; i++ {
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
		if r.Occupants < occMin {
			occMin = r.Occupants
		}
	}
	// The room is busy (~full); over a flat window it should barely move. The old
	// jitter-every-tick model produced >150 enters and >150 leaves here.
	if churn := (r.EnterTotal - enter0) + (r.LeaveTotal - leave0); churn > 15 {
		t.Errorf("enter+leave churn = %d over a flat 75min window, want small (steady room)", churn)
	}
	if occMin < r.MaxOccupancy-5 {
		t.Errorf("occupancy dipped to %d (max %d); steady busy room should not flicker far", occMin, r.MaxOccupancy)
	}
}

func TestTick_AcceleratedStability(t *testing.T) {
	// Large dt (24 min/tick, as produced by timeMultiplier=288 with a 5s tick)
	// must not destabilise the model: values stay bounded over a simulated 48h.
	b := newTestBuilding(monday)
	now := monday
	const dt = 24 * time.Minute
	for i := 0; i < 120; i++ { // 120 * 24min = 48h
		now = now.Add(dt)
		b.Tick(now, dt)
		r := b.room()
		if r.Occupants < 0 || r.Occupants > r.MaxOccupancy {
			t.Fatalf("tick %d: Occupants = %d out of range", i, r.Occupants)
		}
		if math.IsNaN(b.DemandW) || math.IsInf(b.DemandW, 0) || b.DemandW < testBaseLoadW {
			t.Fatalf("tick %d: DemandW = %g invalid", i, b.DemandW)
		}
		if r.LightLevel < 0 || r.LightLevel > 100 || r.FanPct < 0 || r.FanPct > 100 {
			t.Fatalf("tick %d: light=%g fan=%g out of range", i, r.LightLevel, r.FanPct)
		}
	}
}

func TestDaylight(t *testing.T) {
	if d := daylight(at(monday, 3, 0)); d != 0 {
		t.Errorf("daylight 03:00 = %g, want 0", d)
	}
	if d := daylight(at(monday, 22, 0)); d != 0 {
		t.Errorf("daylight 22:00 = %g, want 0", d)
	}
	noon := daylight(at(monday, 13, 0))
	if noon < 0.95 {
		t.Errorf("daylight 13:00 = %g, want near peak", noon)
	}
	if morning := daylight(at(monday, 9, 0)); morning <= 0 || morning >= noon {
		t.Errorf("daylight 09:00 = %g, want between 0 and noon peak %g", morning, noon)
	}
}
