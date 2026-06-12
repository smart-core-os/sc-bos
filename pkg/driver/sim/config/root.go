// Package config defines the configuration schema for the sim driver.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/block/mdblock"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// Root is the top-level configuration for the sim driver.
//
// Unlike the mock driver, which lists every device explicitly, the sim driver is
// configured at the building level: floors, the rooms on each floor, and the
// device "archetypes" present in each room. The driver expands archetypes into
// concrete devices and runs a single coupled simulation across them.
type Root struct {
	driver.BaseConfig
	// Metadata is announced on the driver's own name (BaseConfig.Name).
	Metadata *metadatapb.Metadata `json:"metadata,omitempty"`

	// NamePrefix is prepended to every generated device name, e.g. "van/uk/brum/ugs".
	NamePrefix string `json:"namePrefix,omitempty"`

	// WorkingHours controls the daily occupancy/activity curve. Defaults to 08:00–18:00 Mon–Fri.
	WorkingHours *WorkingHours `json:"workingHours,omitempty"`

	// TimeMultiplier accelerates simulated time relative to the wall clock.
	// 1 (default) runs in real time; 288 compresses a 24h day into 5 minutes.
	TimeMultiplier float64 `json:"timeMultiplier,omitempty"`

	// TickInterval is the wall-clock cadence at which the simulation advances. Defaults to 5s.
	TickInterval *jsontypes.Duration `json:"tickInterval,omitempty"`

	// BaseLoadW is the always-on building electrical load in watts (lifts, servers, etc.). Defaults to 5000.
	BaseLoadW float64 `json:"baseLoadW,omitempty"`

	// Seed seeds the simulation's random source. 0 (default) uses a fixed seed for reproducible demos.
	Seed int64 `json:"seed,omitempty"`

	Floors []Floor `json:"floors,omitempty"`

	// BuildingDevices are devices that report whole-building aggregates rather than
	// belonging to a single room, e.g. the main incoming electricity meter.
	// Only the meter and electric archetypes are meaningful here.
	BuildingDevices []Archetype `json:"buildingDevices,omitempty"`

	// HealthCheck optionally simulates driver-level connectivity faults.
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
}

// WorkingHours describes the building's occupied period.
type WorkingHours struct {
	// Start and End are hours of the day in 24h decimal form, e.g. 8.5 == 08:30.
	Start float64 `json:"start,omitempty"`
	End   float64 `json:"end,omitempty"`
	// Days lists the working weekdays as names ("Mon".."Sun" or "Monday".."Sunday").
	// Defaults to Monday–Friday.
	Days []string `json:"days,omitempty"`
}

// Floor is a level of the building containing rooms.
type Floor struct {
	Name  string `json:"name"`
	Title string `json:"title,omitempty"`
	// MaxOccupancy is the peak number of people on this floor. If a room does not
	// set its own MaxOccupancy, the floor's value is shared evenly between its rooms.
	MaxOccupancy int    `json:"maxOccupancy,omitempty"`
	Rooms        []Room `json:"rooms,omitempty"`
}

// Room is a space within a floor containing device archetypes.
type Room struct {
	Name         string      `json:"name"`
	Title        string      `json:"title,omitempty"`
	MaxOccupancy int         `json:"maxOccupancy,omitempty"`
	Archetypes   []Archetype `json:"archetypes,omitempty"`
}

// Archetype is a class of device present in a room (or the building).
// The driver expands it into Count concrete devices with the traits implied by Type.
type Archetype struct {
	// Type selects the device kind. See the sim package for supported values
	// (lighting, fcu, pir, motion, brightness, airquality, meter, electric, enterleave).
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	// Count is the number of devices to generate. Defaults to 1.
	Count int `json:"count,omitempty"`
	// RatedPowerW is the per-device power draw at full load, in watts.
	// Used by lighting and fcu archetypes to derive building energy use. Sensible defaults apply.
	RatedPowerW float64 `json:"ratedPowerW,omitempty"`
	// SetPointC is the target temperature for fcu archetypes, in Celsius. Defaults to 21.
	SetPointC float64 `json:"setPointC,omitempty"`
}

// HealthCheck configures a simulated driver-level connectivity health check for UI testing.
type HealthCheck struct {
	// FaultProbability is the probability (0.0–1.0) that each evaluation results in a fault state.
	// Defaults to 0.15 when omitted; set it explicitly to 0 to never fault (always healthy).
	FaultProbability *float64 `json:"faultProbability,omitempty"`
}

// Defaults applied during normalisation.
const (
	DefaultTimeMultiplier = 1.0
	DefaultTickInterval   = 5 * time.Second
	DefaultBaseLoadW      = 5000.0
	DefaultWorkStart      = 8.0
	DefaultWorkEnd        = 18.0
	DefaultSeed           = 1
	// DefaultFaultProbability is applied when healthCheck is present but its
	// faultProbability is omitted. An explicit 0 is honoured (never fault).
	DefaultFaultProbability = 0.15
)

// Normalise fills in defaults on a parsed Root. It is safe to call on a zero value.
func (r *Root) Normalise() {
	if r.TimeMultiplier <= 0 {
		r.TimeMultiplier = DefaultTimeMultiplier
	}
	if r.TickInterval == nil || r.TickInterval.Duration <= 0 {
		r.TickInterval = &jsontypes.Duration{Duration: DefaultTickInterval}
	}
	if r.BaseLoadW <= 0 {
		r.BaseLoadW = DefaultBaseLoadW
	}
	if r.Seed == 0 {
		r.Seed = DefaultSeed
	}
	if r.WorkingHours == nil {
		r.WorkingHours = &WorkingHours{Start: DefaultWorkStart, End: DefaultWorkEnd}
	}
	if r.WorkingHours.End <= r.WorkingHours.Start {
		r.WorkingHours.Start = DefaultWorkStart
		r.WorkingHours.End = DefaultWorkEnd
	}
	if r.HealthCheck != nil && r.HealthCheck.FaultProbability == nil {
		p := DefaultFaultProbability
		r.HealthCheck.FaultProbability = &p
	}
}

// Validate checks values that cannot be defaulted. Call it after Normalise.
func (r *Root) Validate() error {
	if r.HealthCheck != nil && r.HealthCheck.FaultProbability != nil {
		if p := *r.HealthCheck.FaultProbability; p < 0 || p > 1 {
			return fmt.Errorf("healthCheck.faultProbability must be between 0 and 1, got %g", p)
		}
	}
	// Generated device names embed the floor and room names, so duplicates would
	// expand to colliding device names.
	floorNames := make(map[string]bool, len(r.Floors))
	for fi, f := range r.Floors {
		if f.Name == "" {
			return fmt.Errorf("floors[%d] has no name", fi)
		}
		if floorNames[f.Name] {
			return fmt.Errorf("duplicate floor name %q", f.Name)
		}
		floorNames[f.Name] = true
		roomNames := make(map[string]bool, len(f.Rooms))
		for ri, room := range f.Rooms {
			if room.Name == "" {
				return fmt.Errorf("floor %q rooms[%d] has no name", f.Name, ri)
			}
			if roomNames[room.Name] {
				return fmt.Errorf("duplicate room name %q on floor %q", room.Name, f.Name)
			}
			roomNames[room.Name] = true
		}
	}
	return nil
}

// Weekdays resolves the configured working day names to time.Weekday values.
// Returns Monday–Friday if none are configured. Unknown names produce an error.
func (w *WorkingHours) Weekdays() ([]time.Weekday, error) {
	if w == nil || len(w.Days) == 0 {
		return []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}, nil
	}
	out := make([]time.Weekday, 0, len(w.Days))
	for _, d := range w.Days {
		wd, ok := weekdayByName[strings.ToLower(strings.TrimSpace(d))]
		if !ok {
			return nil, fmt.Errorf("unknown working day %q", d)
		}
		out = append(out, wd)
	}
	return out, nil
}

var weekdayByName = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "weds": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

// Blocks describes the config structure for the configuration UI.
var Blocks = []block.Block{
	{
		Path: []string{"floors"},
		Key:  "name",
		Blocks: []block.Block{
			{
				Path: []string{"rooms"},
				Key:  "name",
				Blocks: []block.Block{
					{Path: []string{"archetypes"}, Key: "type"},
				},
			},
		},
	},
	{
		Path: []string{"buildingDevices"},
		Key:  "type",
	},
	{
		Path:   []string{"metadata"},
		Blocks: mdblock.Categories,
	},
}
