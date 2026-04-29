package main

import (
	"math"
	"math/rand"
	"time"
)

// OfficeProfile holds all the parameters that describe a building's usage and sensor characteristics.
// It drives both the time-of-day load curve and the realistic ranges for each trait.
// New profiles can be added to the Profiles map to model different building types.
type OfficeProfile struct {
	Schedule    ScheduleProfile
	Occupancy   OccupancyProfile
	Meter       MeterProfile
	Electric    ElectricProfile
	Temperature TemperatureProfile
	AirQuality  AirQualityProfile
	Sound       SoundProfile
	Allocation  AllocationProfile
}

// ScheduleProfile defines when the building is occupied and how quickly readings are sampled.
type ScheduleProfile struct {
	WeekendLoad  float64 // baseline load on weekends (0–1); e.g. 0.05 for a quiet Saturday
	WorkStart    float64 // hour (0–24) when staff begin arriving
	PeakStart    float64 // hour when full occupancy begins
	LunchStart   float64 // hour when lunch dip begins
	LunchEnd     float64 // hour when post-lunch peak resumes (dip bottom is at midpoint)
	WorkEnd      float64 // hour when staff start leaving
	CloseTime    float64 // hour when the building is effectively empty
	PeakLoad     float64 // load factor at full occupancy (0–1)
	LunchDipLoad float64 // load factor at the bottom of the lunch dip (0–1)

	// Sampling intervals at different load levels.
	QuietInterval [2]time.Duration // [min, max] when load < QuietThreshold
	BusyInterval  [2]time.Duration // [min, max] when load >= BusyThreshold
	// Between the two thresholds the quiet range is used.
	QuietThreshold float64
	BusyThreshold  float64
}

// OccupancyProfile defines occupancy sensor characteristics.
type OccupancyProfile struct {
	MaxPeople int // maximum people count at full load
}

// MeterProfile defines cumulative meter characteristics.
type MeterProfile struct {
	PeakPowerKW   float64 // electrical load at full occupancy (kW)
	StandbyFactor float64 // fraction of peak power drawn when the building is empty
}

// ElectricProfile defines electric demand sensor characteristics.
type ElectricProfile struct {
	NightCurrentA float32 // baseline current when empty (HVAC/servers always on)
	PeakCurrentA  float32 // additional current at full occupancy
	VoltageMin    float32
	VoltageMax    float32
	PFBase        float32 // power factor at zero load
	PFRange       float32 // additional power factor at full load (PF = PFBase + PFRange*load)
}

// TemperatureProfile defines air temperature sensor and HVAC characteristics.
type TemperatureProfile struct {
	OccupiedMin       float64       // comfort setpoint lower bound (°C)
	OccupiedMax       float64       // comfort setpoint upper bound (°C)
	SetbackMin        float64       // unoccupied (night/weekend) setback lower bound (°C)
	SetbackMax        float64       // unoccupied setback upper bound (°C)
	OccupiedThreshold float64       // load above which the comfort setpoint applies
	HVACRatePerStep   float64       // max °C the setpoint can change per interval (HVAC ramp speed)
	ThermalLagFactor  float64       // fraction of setpoint gap closed per interval (0–1)
	AmbientNoiseSigma float64       // std dev of random ambient noise (°C)
	Interval          time.Duration // fixed sampling interval
}

// AirQualityProfile defines air quality sensor characteristics.
type AirQualityProfile struct {
	CO2Baseline   float64 // ppm at zero occupancy (outdoor background)
	CO2PeakAbove  float64 // additional ppm at full occupancy
	CO2VentBase   float64 // base approach rate toward CO2 target (exponential smoothing factor)
	CO2VentRange  float64 // additional approach rate at full load (ventilation increases with occupancy)
	VOCBaseline   float64 // µg/m³ at zero occupancy
	VOCPeakAbove  float64 // additional µg/m³ at full occupancy
	VOCApproachRate float64 // approach rate toward VOC target

	PressureHPa    float64 // mean air pressure (hPa)
	PressureNoise  float64 // std dev of pressure variation (hPa)
	AirChangeBase  float64 // air changes per hour at zero occupancy
	AirChangeRange float64 // additional ACH at full occupancy

	ComfortScoreThreshold float32 // score below which comfort is UNCOMFORTABLE

	Interval time.Duration // fixed sampling interval
}

// SoundProfile defines sound sensor characteristics.
type SoundProfile struct {
	NightDB    float64 // dB at zero occupancy (HVAC hum baseline)
	PeakDB     float64 // dB at full occupancy (busy office)
	WalkFactor float64 // how quickly the sound level tracks the load target (0–1)
	NoiseSigma float64 // random walk std dev (dB)
	ClampMin   float64 // minimum plausible dB reading
	ClampMax   float64 // maximum plausible dB reading
}

// AllocationProfile defines desk/room allocation sensor characteristics.
type AllocationProfile struct {
	MaxProbability float64 // probability of an ALLOCATED event at full load (0–1)
}

// Load returns a factor in [0.0, 1.0] representing how busy the building is at time t.
// 0.0 = empty, 1.0 = full peak occupancy.
func (p *OfficeProfile) Load(t time.Time) float64 {
	s := p.Schedule
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return s.WeekendLoad
	}
	hour := float64(t.Hour()) + float64(t.Minute())/60.0
	lunchMid := (s.LunchStart + s.LunchEnd) / 2
	switch {
	case hour < s.WorkStart:
		return 0.0
	case hour < s.PeakStart:
		return lerp(hour, s.WorkStart, s.PeakStart, 0.0, s.PeakLoad)
	case hour < s.LunchStart:
		return s.PeakLoad
	case hour < lunchMid:
		return lerp(hour, s.LunchStart, lunchMid, s.PeakLoad, s.LunchDipLoad)
	case hour < s.LunchEnd:
		return lerp(hour, lunchMid, s.LunchEnd, s.LunchDipLoad, s.PeakLoad)
	case hour < s.WorkEnd:
		return s.PeakLoad
	case hour < s.CloseTime:
		return lerp(hour, s.WorkEnd, s.CloseTime, s.PeakLoad, 0.0)
	default:
		return 0.0
	}
}

// IntervalForLoad returns a sampling interval that is shorter when the building is active.
func (p *OfficeProfile) IntervalForLoad(load float64) time.Duration {
	s := p.Schedule
	lo, hi := s.QuietInterval[0], s.QuietInterval[1]
	if load >= s.BusyThreshold {
		lo, hi = s.BusyInterval[0], s.BusyInterval[1]
	}
	r := int((hi - lo) / time.Minute)
	if r <= 0 {
		return lo
	}
	return lo + time.Duration(rand.Intn(r))*time.Minute
}

// lerp maps v from [fromLow, fromHigh] to [toLow, toHigh].
func lerp(v, fromLow, fromHigh, toLow, toHigh float64) float64 {
	return (v-fromLow)/(fromHigh-fromLow)*(toHigh-toLow) + toLow
}

// clampFloat64 clamps v to [lo, hi].
func clampFloat64(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

// Profiles is the registry of named building profiles selectable via --profile.
var Profiles = map[string]*OfficeProfile{
	"office": officeProfile,
}

// officeProfile models a typical commercial office building (Mon–Fri, 9–5).
var officeProfile = &OfficeProfile{
	Schedule: ScheduleProfile{
		WeekendLoad:    0.05,
		WorkStart:      7.0,
		PeakStart:      9.0,
		LunchStart:     12.0,
		LunchEnd:       14.0,
		WorkEnd:        17.0,
		CloseTime:      19.0,
		PeakLoad:       0.9,
		LunchDipLoad:   0.45,
		QuietInterval:  [2]time.Duration{30 * time.Minute, 61 * time.Minute},
		BusyInterval:   [2]time.Duration{5 * time.Minute, 21 * time.Minute},
		QuietThreshold: 0.1,
		BusyThreshold:  0.5,
	},
	Occupancy: OccupancyProfile{
		MaxPeople: 49,
	},
	Meter: MeterProfile{
		PeakPowerKW:   50.0,
		StandbyFactor: 0.05,
	},
	Electric: ElectricProfile{
		NightCurrentA: 4,
		PeakCurrentA:  36,
		VoltageMin:    238,
		VoltageMax:    243,
		PFBase:        0.75,
		PFRange:       0.20,
	},
	Temperature: TemperatureProfile{
		OccupiedMin:       21.0,
		OccupiedMax:       22.0,
		SetbackMin:        16.0,
		SetbackMax:        18.0,
		OccupiedThreshold: 0.2,
		HVACRatePerStep:   0.5,
		ThermalLagFactor:  0.3,
		AmbientNoiseSigma: 0.5,
		Interval:          15 * time.Minute,
	},
	AirQuality: AirQualityProfile{
		CO2Baseline:           400,
		CO2PeakAbove:          800,
		CO2VentBase:           0.1,
		CO2VentRange:          0.3,
		VOCBaseline:           50,
		VOCPeakAbove:          350,
		VOCApproachRate:       0.15,
		PressureHPa:           1013,
		PressureNoise:         3,
		AirChangeBase:         2,
		AirChangeRange:        4,
		ComfortScoreThreshold: 70,
		Interval:              15 * time.Minute,
	},
	Sound: SoundProfile{
		NightDB:    22,
		PeakDB:     60,
		WalkFactor: 0.2,
		NoiseSigma: 2.0,
		ClampMin:   15,
		ClampMax:   75,
	},
	Allocation: AllocationProfile{
		MaxProbability: 0.8,
	},
}
