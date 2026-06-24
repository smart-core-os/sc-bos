package sim

import (
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/sim/scale"
)

// errBusy is returned by submit when an override cannot be enqueued.
var errBusy = errors.New("sim: engine busy, command dropped")

// Building is the coupled simulation engine. A single goroutine owns it: Tick
// advances the physical state (occupancy → lights/FCUs → energy; daylight → brightness;
// occupancy → CO2), and the registered Updaters then publish the relevant slice of
// that state into each device's gRPC trait model.
//
// All state mutation happens on the engine goroutine, so the struct needs no locking;
// the underlying trait models are independently goroutine-safe for concurrent readers.
type Building struct {
	scaler    scale.Time
	baseLoadW float64
	rng       *rand.Rand

	Floors []*Floor

	// Aggregates, recomputed each Tick.
	DemandW   float64   // instantaneous whole-building electrical demand, watts
	MeterKWh  float64   // monotonically accumulating energy use, kWh
	meterFrom time.Time // start of the metering period
	day       float64   // outdoor daylight factor (0..1), recomputed each Tick

	updaters []Updater

	// cmds carries override mutations from the MockDeviceApi onto the engine
	// goroutine; Tick drains them before advancing, so all Room state continues to
	// be touched by a single goroutine and needs no locking.
	cmds chan func()
}

// Floor is a level of the building.
type Floor struct {
	Name  string
	Rooms []*Room
}

// Room holds the simulated state of a single space. Field values are the
// authoritative simulation state that device updaters read from.
type Room struct {
	Name         string
	MaxOccupancy int

	Occupants  int
	LightLevel float64 // 0..100 %
	FanPct     float64 // 0..100 %
	TempC      float64
	SetPointC  float64
	CO2ppm     float64
	EnterTotal int64
	LeaveTotal int64

	// Overrides set via the MockDeviceApi. While non-nil the engine uses the
	// override in place of the simulated input and the coupled state responds to
	// it (e.g. a forced occupancy still drives lighting, FCU load, CO2 and power).
	// A nil override means the simulation drives the value as normal.
	occupancyOverride *int
	setPointOverride  *float64
	lightOverride     *float64
	fanOverride       *float64

	// occupancyF is the continuous occupancy that Occupants is rounded from, and
	// occNoise is a slow, mean-reverting noise term. Together they let the
	// population wander organically without manufacturing enter/leave churn every
	// tick (re-jittering the target each tick would flip the rounded count
	// constantly even when the real occupancy is steady).
	occupancyF float64
	occNoise   float64

	// rated capacities, summed from the room's lighting/fcu archetypes.
	ratedLightingW float64
	ratedFcuW      float64
	// per-tick derived loads.
	lightsW float64
	fcuW    float64
}

// NewBuilding constructs the engine from pre-expanded floor/room state. start is
// the simulated time the building begins at; rooms are initialised to a plausible
// steady state for that time so devices report sensible values from the first tick.
func NewBuilding(start time.Time, scaler scale.Time, baseLoadW float64, seed int64, floors []*Floor) *Building {
	b := &Building{
		scaler:    scaler,
		baseLoadW: baseLoadW,
		rng:       rand.New(rand.NewSource(seed)),
		Floors:    floors,
		meterFrom: start,
		cmds:      make(chan func(), 64),
	}
	work := scaler.At(start)
	day := daylight(start)
	b.day = day
	b.DemandW = baseLoadW
	for _, f := range b.Floors {
		for _, r := range f.Rooms {
			r.reset(work, day)
			b.DemandW += r.lightsW + r.fcuW
		}
	}
	return b
}

// AddUpdater registers u to be invoked after each Tick.
func (b *Building) AddUpdater(u Updater) {
	b.updaters = append(b.updaters, u)
}

// submit enqueues fn to run on the engine goroutine at the start of the next Tick.
// It never blocks: if the queue is full (or the engine has stopped draining it)
// the command is dropped and an error returned, which the caller surfaces to the
// API client rather than stalling the gRPC handler.
func (b *Building) submit(fn func()) error {
	select {
	case b.cmds <- fn:
		return nil
	default:
		return errBusy
	}
}

// drainCommands applies all queued override mutations. Called at the top of Tick
// so overrides take effect on the engine goroutine that owns the Room state.
func (b *Building) drainCommands() {
	for {
		select {
		case fn := <-b.cmds:
			fn()
		default:
			return
		}
	}
}

// Tick advances the simulation by dt of simulated time, with now being the new
// simulated instant. dt may be zero (e.g. the priming tick), in which case state
// targets are approached by zero and the meter does not accumulate.
func (b *Building) Tick(now time.Time, dt time.Duration) {
	b.drainCommands()
	work := b.scaler.At(now)
	b.day = daylight(now)

	b.DemandW = b.baseLoadW
	for _, f := range b.Floors {
		for _, r := range f.Rooms {
			r.tick(work, b.day, dt, b.rng)
			b.DemandW += r.lightsW + r.fcuW
		}
	}
	if dt > 0 {
		b.MeterKWh += b.DemandW / 1000 * dt.Hours()
	}
}

// Publish invokes every registered updater with the current state.
func (b *Building) Publish(now time.Time) {
	for _, u := range b.updaters {
		u.Update(now, b)
	}
}

// roomTargets computes the steady-state values a room tends toward for the given
// time-of-day work factor (0..1), outdoor daylight factor (0..1) and effective
// temperature set point.
func roomTargets(r *Room, work, day, setPoint float64) (light, fan, temp, co2 float64) {
	occRatio := 0.0
	if r.MaxOccupancy > 0 {
		occRatio = float64(r.Occupants) / float64(r.MaxOccupancy)
	}

	// Lights: on when the room is occupied or during core hours; dimmer when there's
	// more daylight coming through the windows.
	if r.Occupants > 0 || work > 0.3 {
		light = 85 - day*30
		if light < 20 {
			light = 20
		}
	}

	// FCU: idles at a low baseline while the building is open, ramping with occupancy.
	if work > 0 || r.Occupants > 0 {
		fan = 20 + occRatio*80
	}

	// Temperature: occupants add heat; the FCU fan offsets part of it but can't
	// fully cancel it, so a busier room is always warmer. An empty or idle room
	// settles back at the setpoint (the offset is floored at zero).
	heat := occRatio * 4     // occupant heat gain, up to +4°C at full occupancy
	cooling := fan / 100 * 2 // the fan removes up to 2°C
	temp = setPoint + max(0, heat-cooling)

	// CO2: ~420ppm baseline outdoors, rising with how full the room is.
	co2 = 420 + occRatio*1200
	return light, fan, temp, co2
}

// reset assigns a room's steady-state values directly (used at construction).
func (r *Room) reset(work, day float64) {
	if r.SetPointC == 0 {
		r.SetPointC = defaultSetPointC
	}
	r.occupancyF = work * float64(r.MaxOccupancy)
	r.Occupants = int(math.Round(r.occupancyF))
	light, fan, temp, co2 := roomTargets(r, work, day, r.SetPointC)
	r.LightLevel, r.FanPct, r.TempC, r.CO2ppm = light, fan, temp, co2
	r.recomputeLoads()
}

// tick advances a room toward its targets over dt, updating occupancy first (which
// the other targets depend on) and tracking enter/leave totals.
func (r *Room) tick(work, day float64, dt time.Duration, rng *rand.Rand) {
	prev := r.Occupants
	if r.occupancyOverride != nil {
		// Held occupancy: skip the organic drift and pin the count to the forced
		// value. The other targets below still derive from it, so the rest of the
		// room (and the building aggregates) respond to the forced occupancy.
		r.Occupants = *r.occupancyOverride
		r.occupancyF = float64(r.Occupants)
	} else {
		// Occupancy drifts toward the time-of-day target. occNoise is a slow,
		// mean-reverting noise term (a low-pass of fresh ±5% samples) so the population
		// wanders organically; occupancyF is the continuous value the integer count is
		// derived from. Smoothing both means the target moves gradually instead of
		// teleporting every tick.
		r.occNoise = approach(r.occNoise, rng.Float64()*0.1-0.05, 0.2, dt)
		target := work * float64(r.MaxOccupancy) * (1 + r.occNoise)
		r.occupancyF = approach(r.occupancyF, target, 0.5, dt)

		// Update the integer count with hysteresis: it only moves once occupancyF has
		// crossed more than half a person past the current count (plus a small margin),
		// so sub-person jitter doesn't manufacture enter/leave events while the room is
		// essentially steady. Genuine moves (the day's occupancy arc) still re-round.
		const hysteresis = 0.5 + 0.2
		if d := r.occupancyF - float64(prev); d > hysteresis || d < -hysteresis {
			r.Occupants = int(math.Round(r.occupancyF))
		}
	}
	if r.Occupants < 0 {
		r.Occupants = 0
	}
	if r.Occupants > r.MaxOccupancy {
		r.Occupants = r.MaxOccupancy
	}
	if d := r.Occupants - prev; d > 0 {
		r.EnterTotal += int64(d)
	} else if d < 0 {
		r.LeaveTotal += int64(-d)
	}

	light, fan, temp, co2 := roomTargets(r, work, day, r.setPoint())
	// A forced light/fan pins the actuator directly; otherwise it eases toward the
	// occupancy-driven target. Either way the loads (and thus building demand and
	// metered energy) recompute from the resulting levels below.
	if r.lightOverride != nil {
		r.LightLevel = *r.lightOverride
	} else {
		r.LightLevel = approach(r.LightLevel, light, 1.0, dt)
	}
	if r.fanOverride != nil {
		r.FanPct = *r.fanOverride
	} else {
		r.FanPct = approach(r.FanPct, fan, 0.5, dt)
	}
	r.TempC = approach(r.TempC, temp, 0.3, dt)
	r.CO2ppm = approach(r.CO2ppm, co2, 0.4, dt)
	r.recomputeLoads()
}

func (r *Room) recomputeLoads() {
	r.lightsW = r.LightLevel / 100 * r.ratedLightingW
	r.fcuW = r.FanPct / 100 * r.ratedFcuW
}

// roomOverride identifies which simulated input a forced value drives.
type roomOverride int

const (
	overrideOccupancy roomOverride = iota
	overrideSetPoint
	overrideLight
	overrideFan
)

// setPoint returns the effective temperature set point: the forced value if one
// is held, otherwise the room's configured set point.
func (r *Room) setPoint() float64 {
	if r.setPointOverride != nil {
		return *r.setPointOverride
	}
	return r.SetPointC
}

// set applies a forced value to the given input. Called only on the engine
// goroutine (via Building.drainCommands).
func (r *Room) set(o roomOverride, v float64) {
	switch o {
	case overrideOccupancy:
		n := int(math.Round(v))
		if n < 0 {
			n = 0
		}
		r.occupancyOverride = &n
	case overrideSetPoint:
		r.setPointOverride = &v
	case overrideLight:
		l := clampPct(v)
		r.lightOverride = &l
	case overrideFan:
		f := clampPct(v)
		r.fanOverride = &f
	}
}

// hold freezes an input at its current simulated value (SetDeviceAutomation off).
func (r *Room) hold(o roomOverride) {
	switch o {
	case overrideOccupancy:
		r.set(overrideOccupancy, float64(r.Occupants))
	case overrideSetPoint:
		r.set(overrideSetPoint, r.setPoint())
	case overrideLight:
		r.set(overrideLight, r.LightLevel)
	case overrideFan:
		r.set(overrideFan, r.FanPct)
	}
}

// release clears an override so the simulation drives the input again
// (SetDeviceAutomation on).
func (r *Room) release(o roomOverride) {
	switch o {
	case overrideOccupancy:
		r.occupancyOverride = nil
	case overrideSetPoint:
		r.setPointOverride = nil
	case overrideLight:
		r.lightOverride = nil
	case overrideFan:
		r.fanOverride = nil
	}
}

func clampPct(v float64) float64 {
	return min(max(v, 0), 100)
}

// approach moves cur a fraction of the way toward target. The fraction is
// ratePerMin scaled by dt and clamped to [0,1], which keeps the step stable for
// any dt — including the large dt values produced under time acceleration.
func approach(cur, target, ratePerMin float64, dt time.Duration) float64 {
	f := min(max(ratePerMin*dt.Minutes(), 0), 1)
	return cur + (target-cur)*f
}

// daylight returns the outdoor daylight factor (0..1) for the time of day,
// a smooth bump that is zero before sunrise and after sunset and peaks at midday.
func daylight(t time.Time) float64 {
	const sunrise, sunset = 6.0, 20.0
	h := float64(t.Hour()) + float64(t.Minute())/60.0
	if h <= sunrise || h >= sunset {
		return 0
	}
	x := (h - sunrise) / (sunset - sunrise) // 0..1 across daylight hours
	return math.Sin(x * math.Pi)
}
