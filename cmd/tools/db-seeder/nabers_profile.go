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
}
