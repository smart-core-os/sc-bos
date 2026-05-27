package main

import "time"

// nabersSchedule is a shared office schedule used by all NABERS profiles.
// Hourly intervals are sufficient for NABERS reporting (minimum requirement is quarterly).
var nabersSchedule = ScheduleProfile{
	WeekendLoad:    0.05,
	WorkStart:      7.0,
	PeakStart:      9.0,
	LunchStart:     12.0,
	LunchEnd:       14.0,
	WorkEnd:        17.0,
	CloseTime:      19.0,
	PeakLoad:       0.9,
	LunchDipLoad:   0.45,
	QuietInterval:  [2]time.Duration{time.Hour, time.Hour},
	BusyInterval:   [2]time.Duration{time.Hour, time.Hour},
	QuietThreshold: 0.1,
	BusyThreshold:  0.5,
}

func init() {
	Profiles["nabers-excellent"] = &OfficeProfile{
		NabersOnly: true,
		Schedule:   nabersSchedule,
		// ~5.5–5.7 stars: good building, just above the DfP target.
		// PeakPowerKW=50 with 8% standby → ~59 kWh/m²/yr net across base building meters
		// (6 consumption meters minus 1 PV, all sharing this profile).
		Meter: MeterProfile{
			PeakPowerKW:   50,
			StandbyFactor: 0.08,
		},
	}

	Profiles["nabers-ok"] = &OfficeProfile{
		NabersOnly: true,
		Schedule:   nabersSchedule,
		// ~3 stars: market average building.
		// PeakPowerKW=80 with 8% standby ≈ 190 kWh/m²/yr for a medium office.
		Meter: MeterProfile{
			PeakPowerKW:   80,
			StandbyFactor: 0.08,
		},
	}

	Profiles["nabers-poor"] = &OfficeProfile{
		NabersOnly: true,
		Schedule:   nabersSchedule,
		// ~1–2 stars: high consumption, significant standby waste.
		// PeakPowerKW=150 with 15% standby ≈ 320 kWh/m²/yr for a medium office.
		Meter: MeterProfile{
			PeakPowerKW:   150,
			StandbyFactor: 0.15,
		},
	}

	// nabers-*-env profiles seed only environmental/sensor traits (air quality, electric,
	// temperature, sound, occupancy, allocation) without meter data. Use these alongside
	// the nabers-* meter-only profiles to separately seed each data category.

	Profiles["nabers-excellent-env"] = &OfficeProfile{
		SkipMeter: true,
		Schedule:  nabersSchedule,
		// Well-managed building: low CO2, tight temperature control, quiet.
		Occupancy: OccupancyProfile{MaxPeople: 49},
		Electric: ElectricProfile{
			NightCurrentA: 4,
			PeakCurrentA:  36,
			VoltageMin:    238,
			VoltageMax:    243,
			PFBase:        0.75,
			PFRange:       0.20,
		},
		Temperature: TemperatureProfile{
			OccupiedMin:       21.5,
			OccupiedMax:       22.0,
			SetbackMin:        17.0,
			SetbackMax:        18.0,
			OccupiedThreshold: 0.2,
			HVACRatePerStep:   0.5,
			ThermalLagFactor:  0.3,
			AmbientNoiseSigma: 0.3,
			Interval:          15 * time.Minute,
		},
		AirQuality: AirQualityProfile{
			CO2Baseline:           400,
			CO2PeakAbove:          400, // well-ventilated: CO2 stays low even at peak
			CO2VentBase:           0.2,
			CO2VentRange:          0.5,
			VOCBaseline:           30,
			VOCPeakAbove:          150,
			VOCApproachRate:       0.2,
			PressureHPa:           1013,
			PressureNoise:         2,
			AirChangeBase:         3,
			AirChangeRange:        5,
			ComfortScoreThreshold: 70,
			Interval:              15 * time.Minute,
		},
		Sound: SoundProfile{
			NightDB:    20,
			PeakDB:     52,
			WalkFactor: 0.2,
			NoiseSigma: 1.5,
			ClampMin:   15,
			ClampMax:   65,
		},
		Allocation: AllocationProfile{MaxProbability: 0.8},
	}

	Profiles["nabers-ok-env"] = &OfficeProfile{
		SkipMeter: true,
		Schedule:  nabersSchedule,
		// Average building: moderate CO2 rise, standard temperature band, typical noise.
		Occupancy: OccupancyProfile{MaxPeople: 49},
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
		Allocation: AllocationProfile{MaxProbability: 0.8},
	}

	Profiles["nabers-poor-env"] = &OfficeProfile{
		SkipMeter: true,
		Schedule:  nabersSchedule,
		// Poorly managed building: high CO2, loose temperature control, noisy.
		Occupancy: OccupancyProfile{MaxPeople: 49},
		Electric: ElectricProfile{
			NightCurrentA: 8,
			PeakCurrentA:  50,
			VoltageMin:    235,
			VoltageMax:    245,
			PFBase:        0.68,
			PFRange:       0.18,
		},
		Temperature: TemperatureProfile{
			OccupiedMin:       20.0,
			OccupiedMax:       24.0,
			SetbackMin:        14.0,
			SetbackMax:        20.0,
			OccupiedThreshold: 0.2,
			HVACRatePerStep:   0.3,
			ThermalLagFactor:  0.15,
			AmbientNoiseSigma: 1.2,
			Interval:          15 * time.Minute,
		},
		AirQuality: AirQualityProfile{
			CO2Baseline:           450,
			CO2PeakAbove:          1400, // poor ventilation: CO2 climbs sharply
			CO2VentBase:           0.05,
			CO2VentRange:          0.1,
			VOCBaseline:           80,
			VOCPeakAbove:          600,
			VOCApproachRate:       0.08,
			PressureHPa:           1013,
			PressureNoise:         4,
			AirChangeBase:         1,
			AirChangeRange:        2,
			ComfortScoreThreshold: 70,
			Interval:              15 * time.Minute,
		},
		Sound: SoundProfile{
			NightDB:    28,
			PeakDB:     70,
			WalkFactor: 0.25,
			NoiseSigma: 3.0,
			ClampMin:   20,
			ClampMax:   85,
		},
		Allocation: AllocationProfile{MaxProbability: 0.8},
	}
}
