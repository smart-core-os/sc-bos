package bms

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/multierr"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/auto/bms/config"
)

func processReadState(ctx context.Context, readState *ReadState, writeState *WriteState, actions Actions) (time.Duration, error) {
	now := readState.Now()
	modeUpdates := &modeUpdates{}
	setPointUpdates := make(map[string]float32)
	var ttl ttlVal

	// process occupancy state
	unoccupiedDelay := readState.Config.UnoccupiedDelay.Or(config.DefaultUnoccupiedDelay)
	occupiedCount, totalExpectedOccupancy, noResponseFromSensor, unoccupiedFor := analyseOccupancy(now, readState)
	schedOccupied, occupiedStart, occupiedEnd, occupancySchedChanges := analyseTimeOfDay(now, readState.Config.OccupiedSchedule)
	usingOccupancySched := !occupiedStart.IsZero() || !occupiedEnd.IsZero() || occupancySchedChanges != 0
	if usingOccupancySched {
		ttl.set(occupancySchedChanges)
	}
	turnOccupancyOn := func() {
		for _, target := range readState.Config.OccupancyModeTargets {
			target.ApplyDefaults(config.DefaultOccupancyModeTarget)
			modeUpdates.setMode(target.Name, target.Key, target.On)
		}
	}
	turnOccupancyOff := func() {
		for _, target := range readState.Config.OccupancyModeTargets {
			target.ApplyDefaults(config.DefaultOccupancyModeTarget)
			modeUpdates.setMode(target.Name, target.Key, target.Off)
		}
	}
	switch {
	case occupiedCount > 0:
		if usingOccupancySched {
			if schedOccupied {
				writeState.AddReasonf("occupied within [%s,%s)", occupiedStart.Format("15:04"), occupiedEnd.Format("15:04"))
			} else {
				writeState.AddReasonf("occupancy starts in %v", formatDuration(occupancySchedChanges))
				turnOccupancyOff()
				break
			}
		}
		if noResponseFromSensor > 0 {
			writeState.AddReasonf("%d/%d[/%d] occupied", occupiedCount, totalExpectedOccupancy-noResponseFromSensor, totalExpectedOccupancy)
		} else {
			writeState.AddReasonf("%d/%d occupied", occupiedCount, totalExpectedOccupancy)
		}
		turnOccupancyOn()
	case unoccupiedFor >= unoccupiedDelay:
		if noResponseFromSensor > 0 {
			writeState.AddReasonf("unoccupied for %v %d/%d", formatDuration(unoccupiedFor), totalExpectedOccupancy-noResponseFromSensor, totalExpectedOccupancy)
		} else {
			writeState.AddReasonf("unoccupied for %v", formatDuration(unoccupiedFor))
		}
		turnOccupancyOff()
	case noResponseFromSensor == totalExpectedOccupancy:
		// this case also triggers when no sensors are configured
		writeState.AddReason("no occupancy")
		if !usingOccupancySched {
			break
		}
		if schedOccupied {
			writeState.AddReasonf("occupied within [%s,%s)", occupiedStart.Format("15:04"), occupiedEnd.Format("15:04"))
			turnOccupancyOn()
		} else {
			writeState.AddReasonf("occupancy starts in %v", formatDuration(occupancySchedChanges))
			turnOccupancyOff()
		}
	case unoccupiedFor == 0:
		writeState.AddReasonf("ambiguous occupancy %d/%d", totalExpectedOccupancy-noResponseFromSensor, totalExpectedOccupancy)
	default:
		writeState.AddReasonf("unoccupied %v<%v ago", formatDuration(unoccupiedFor), formatDuration(unoccupiedDelay))
		ttl.set(unoccupiedDelay - unoccupiedFor)
	}

	// deadband adjustment processing
	deadbandOn, onStart, onEnd, deadbandChangesIn := analyseTimeOfDay(now, readState.Config.DeadbandSchedule)
	ttl.set(deadbandChangesIn)
	switch {
	case deadbandOn:
		writeState.AddReasonf("day within [%s,%s)", onStart.Format("15:04"), onEnd.Format("15:04"))
		for _, target := range readState.Config.DeadbandModeTargets {
			modeUpdates.setMode(target.Name, target.Key, target.On)
		}
	case !deadbandOn && deadbandChangesIn == 0:
		// not configured
	case !deadbandOn:
		writeState.AddReasonf("day starts in %v", formatDuration(deadbandChangesIn))
		for _, target := range readState.Config.DeadbandModeTargets {
			modeUpdates.setMode(target.Name, target.Key, target.Off)
		}
	}

	// temperature set point processing
	autoMode, autoSetPoint, autoChangesIn, autoReason := analyseSetPoint(now, readState)
	writeState.AddReason(autoReason) // there's always a reason
	ttl.set(autoChangesIn)
	switch {
	case autoMode:
		if md := readState.Config.ModeSource; md.Name != "" {
			modeUpdates.setMode(md.Name, md.Key, md.On)
		}
		for _, thermostat := range readState.Config.AutoThermostats {
			setPointUpdates[thermostat] = autoSetPoint
		}
	}

	// apply the updates
	// write the modes first
	// then the set points
	err := multierr.Combine(
		modeUpdates.write(ctx, writeState, actions),
		writeSetPointUpdates(ctx, writeState, actions, setPointUpdates),
	)

	return time.Duration(ttl), err
}

// analyseOccupancy inspects the occupancy readings and returns values useful for working out what to do.
// - occupied represents the number of sensors that are currently occupied.
// - total is the total number of sensors we are listening to.
// - noResponse is the number of sensors that we have no response from.
// - unoccupiedFor is the duration since the most recent unoccupied sensor changed to unoccupied.
func analyseOccupancy(now time.Time, state *ReadState) (occupied, total, noResponse int, unoccupiedFor time.Duration) {
	var mostRecentUnoccupiedTime time.Time
	for _, sensor := range state.Config.OccupancySensors {
		total++
		if o, ok := state.Occupancy[sensor]; ok {
			switch o.V.State {
			case traits.Occupancy_OCCUPIED:
				occupied++
			case traits.Occupancy_UNOCCUPIED, traits.Occupancy_IDLE:
				if o.V.StateChangeTime == nil {
					continue
				}
				candidate := o.V.StateChangeTime.AsTime()
				if candidate.After(mostRecentUnoccupiedTime) {
					mostRecentUnoccupiedTime = candidate
				}
			}
		} else {
			noResponse++
		}
	}
	if !mostRecentUnoccupiedTime.IsZero() {
		unoccupiedFor = now.Sub(mostRecentUnoccupiedTime)
	}
	return
}

// analyseTimeOfDay works out if we are currently within a schedule period, and when that might change.
// It returns:
//   - on: true if now is currently within a schedule config.Range
//   - onStart: the next time the matching range starts
//   - onEnd: the next time the matching range ends, aka the end of the matched range
//   - changesIn: the time until the matching schedule range ends, or the next range starts
func analyseTimeOfDay(now time.Time, schedule []config.Range) (on bool, onStart, onEnd time.Time, changesIn time.Duration) {
	checkTime := func(t time.Time) {
		d := t.Sub(now)
		if changesIn == 0 || d < changesIn {
			changesIn = d
		}
	}
	for _, period := range schedule {
		startAt := period.Start.Next(now)
		endAt := period.End.Next(now)
		if startAt.After(endAt) {
			// If start is after end, then we're currently within the range of [start, end).
			// This changesIn calculation falls down if periods overlap, but for simplicity we'll use this simple approach.
			return true, startAt, endAt, endAt.Sub(now)
		}
		checkTime(startAt)
		checkTime(endAt)
	}
	return
}

func analyseSetPoint(now time.Time, state *ReadState) (auto bool, setPoint float32, changesIn time.Duration, reason string) {
	src := state.Config.ModeSource
	if src.Name == "" {
		// assume manual if not configured
		return false, 0, 0, ""
	}
	modeVal, ok := state.Modes[src.Name]
	if !ok {
		return false, 0, 0, "mode not responded"
	}
	val, ok := modeVal[src.Key]
	if !ok {
		return false, 0, 0, fmt.Sprintf("no %s", src.Key)
	}

	autoSetPoint := config.PtrOr(state.Config.AutoModeSetPoint, config.DefaultAutoModeSetPoint)
	if state.MeanOATemp != nil {
		// this is the [CEN Standard BS En 15251] algorithm as outlined in [CIBSE TM52 2013 4.1.4].
		comfortTemp := 0.33*state.MeanOATemp.ValueCelsius + 18.8
		autoSetPoint = float32(comfortTemp)
	}

	switch {
	case val.V == src.On:
		return true, autoSetPoint, 0, "auto mode"
	case val.V == src.Off:
		// has it been off too long
		age := now.Sub(val.At)
		delay := state.Config.ResetModeSourceDelay.Or(config.DefaultResetModeSourceDelay)
		if age >= delay {
			return true, autoSetPoint, 0, fmt.Sprintf("manual set %v>=%v ago", formatDuration(age), formatDuration(delay))
		} else {
			return false, 0, delay - age, fmt.Sprintf("manual set %v<%v ago", formatDuration(age), formatDuration(delay))
		}
	default:
		return false, 0, 0, fmt.Sprintf("mode unknown %q", val.V)
	}
}

func writeSetPointUpdates(ctx context.Context, state *WriteState, actions Actions, updates map[string]float32) error {
	var errs error
	for name, setPoint := range updates {
		req := &traits.UpdateAirTemperatureRequest{
			Name: name,
			State: &traits.AirTemperature{TemperatureGoal: &traits.AirTemperature_TemperatureSetPoint{
				TemperatureSetPoint: &types.Temperature{ValueCelsius: float64(setPoint)},
			}},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"temperature_set_point"}},
		}
		errs = multierr.Append(errs, actions.UpdateAirTemperature(ctx, req, state))
	}
	return errs
}

type modeUpdates map[string]map[string]string

func (m *modeUpdates) setMode(name, key, value string) {
	if m == nil {
		*m = make(map[string]map[string]string)
	}
	if _, ok := (*m)[name]; !ok {
		(*m)[name] = make(map[string]string)
	}
	(*m)[name][key] = value
}

func (m *modeUpdates) write(ctx context.Context, state *WriteState, actions Actions) error {
	var errs error
	for name, modes := range *m {
		req := &traits.UpdateModeValuesRequest{
			Name:       name,
			ModeValues: &traits.ModeValues{Values: modes},
		}
		errs = multierr.Append(errs, actions.UpdateModeValues(ctx, req, state))
	}
	return errs
}

type ttlVal time.Duration

func (t *ttlVal) set(ttl time.Duration) {
	if t == nil || *t == 0 {
		*t = ttlVal(ttl)
	}
	if ttl <= 0 {
		return
	}
	if ttl < time.Duration(*t) {
		*t = ttlVal(ttl)
	}
}
