package setpointhealth

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/smart-core-os/sc-bos/internal/protobuf/protopath2"
	"github.com/smart-core-os/sc-bos/pkg/auto/internal/anytrait"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// setPointEpsilon is the smallest set point change (in native units) treated as a new target.
// Devices report the commanded set point unchanged between adjustments, so this only guards against
// floating point representation jitter.
const setPointEpsilon = 1e-9

// worker watches the measured value and set point of a single device and drives a fault check.
//
// It implements an on-delay: the check only goes abnormal once abs(measured - setPoint) has
// exceeded tolerance continuously, against the same set point, for the configured duration. The
// countdown restarts when the set point changes (the equipment is given a fresh window to reach the
// new target) and is cleared when the deviation returns within tolerance. Once the fault is raised
// it is only cleared by the deviation returning within tolerance — a later set point change cannot
// un-confirm a sustained fault.
//
// When maxDuration is set it acts as a backstop: the check goes abnormal once the deviation has
// exceeded tolerance continuously for maxDuration, regardless of how often the set point changes.
// This catches equipment that never tracks while its set point is repeatedly adjusted, which would
// otherwise keep restarting the per-target countdown and never trip. The backstop clock starts on
// the first breach and is cleared only when the deviation returns within tolerance.
type worker struct {
	check       *healthpb.FaultCheck
	measured    protopath.Path
	setPoint    protopath.Path
	tolerance   float64
	duration    time.Duration
	maxDuration time.Duration // zero disables the backstop
	device      string
	logger      *zap.Logger
}

func (w *worker) run(ctx context.Context, changes <-chan anytrait.Value) error {
	var timer *time.Timer
	var timerC <-chan time.Time
	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			timerC = nil
		}
	}
	defer stopTimer()

	// backstop fires after maxDuration of continuous breach regardless of set point changes. It is
	// started on the first breach and cleared only on a return to tolerance — set point changes do
	// not reset it. Left nil (never started) when maxDuration is zero.
	var backstop *time.Timer
	var backstopC <-chan time.Time
	stopBackstop := func() {
		if backstop != nil {
			backstop.Stop()
			backstop = nil
			backstopC = nil
		}
	}
	defer stopBackstop()

	tripped := false      // whether the check currently reports a fault
	haveSetPoint := false // whether lastSetPoint holds a real reading
	var lastSetPoint float64
	var lastMeasured, lastTarget float64 // remembered for the fault summary when the timer fires

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v, ok := <-changes:
			if !ok {
				return nil
			}

			meas, measSet, measErr := w.extractFloat(w.measured, v.Proto())
			sp, spSet, spErr := w.extractFloat(w.setPoint, v.Proto())
			if err := cmp.Or(measErr, spErr); err != nil {
				// a malformed/non-numeric reading: report unreliable, leave the countdown as-is
				w.logger.Debug("value extraction failed", zap.Error(err))
				w.check.UpdateReliability(ctx, healthpb.ReliabilityFromErr(fmt.Errorf("read %q: %w", w.device, err)))
				continue
			}
			if !measSet || !spSet {
				// measured or set point is unset (a data gap): can't evaluate. Surface as a bad
				// response and freeze the countdown — a gap is not evidence of a control fault.
				w.logger.Debug("measured or set point is unset")
				w.check.UpdateReliability(ctx, healthpb.ReliabilityFromErr(
					fmt.Errorf("measured or set point not set for %q", w.device)))
				stopTimer()
				stopBackstop()
				continue
			}

			setPointChanged := haveSetPoint && math.Abs(sp-lastSetPoint) > setPointEpsilon
			lastSetPoint, haveSetPoint = sp, true
			lastMeasured, lastTarget = meas, sp

			inTolerance := math.Abs(meas-sp) <= w.tolerance
			switch {
			case inTolerance:
				// healthy: cancel any countdown and report normal.
				stopTimer()
				stopBackstop()
				tripped = false
				w.check.ClearFaults() // NORMAL + reliable, idempotent
			case tripped:
				// confirmed fault: refresh the summary with the latest numbers, but keep the fault.
				w.check.SetFault(w.faultErr(meas, sp))
			default:
				// out of tolerance but not yet a confirmed fault: still healthy while we count down.
				w.check.ClearFaults() // NORMAL + reliable, idempotent
				if setPointChanged {
					stopTimer() // new target, fresh window
				}
				if timer == nil {
					timer = time.NewTimer(w.duration)
					timerC = timer.C
				}
				// the backstop ignores set point changes: start it on the first breach and let it run.
				if w.maxDuration > 0 && backstop == nil {
					backstop = time.NewTimer(w.maxDuration)
					backstopC = backstop.C
				}
			}
		case <-timerC:
			timer = nil
			timerC = nil
			stopBackstop() // fault confirmed; backstop no longer needed
			w.check.SetFault(w.faultErr(lastMeasured, lastTarget))
			tripped = true
		case <-backstopC:
			backstop = nil
			backstopC = nil
			stopTimer() // fault confirmed; per-target window no longer needed
			w.check.SetFault(w.faultErr(lastMeasured, lastTarget))
			tripped = true
		}
	}
}

// extractFloat reads the numeric value at rpath from msg.
// The returned bool is false when the field (or a message along its path) is unset, which is a data
// gap rather than an error. A non-nil error means the path resolved to a non-numeric value.
func (w *worker) extractFloat(rpath protopath.Path, msg proto.Message) (float64, bool, error) {
	values, err := protopath2.PathValues(rpath, msg)
	if err != nil {
		return 0, false, err
	}
	if !protopath2.FieldsAreSet(values) {
		return 0, false, nil
	}
	f, ok := floatOf(values.Index(-1).Value)
	if !ok {
		return 0, false, fmt.Errorf("value at %q is not numeric", rpath)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		// devices (notably over BACnet) can report NaN/Inf for an absent or faulted reading;
		// treat it as unreliable rather than a real value that would falsely trip the check.
		return 0, false, fmt.Errorf("value at %q is not a finite number", rpath)
	}
	return f, true, nil
}

// numericLeaf reports an error if path does not end at a numeric scalar field, i.e. one whose value
// floatOf can convert. It lets a misconfigured value path be rejected when the check is created,
// rather than silently reporting every reading as unreliable at runtime.
func numericLeaf(path protopath.Path) error {
	if len(path) == 0 {
		return fmt.Errorf("path is empty")
	}
	last := path[len(path)-1]
	if last.Kind() != protopath.FieldAccessStep {
		return fmt.Errorf("path must end at a scalar field")
	}
	fd := last.FieldDescriptor()
	if fd.IsList() || fd.IsMap() {
		return fmt.Errorf("field %q is repeated or a map, want a numeric scalar", fd.Name())
	}
	switch fd.Kind() {
	case protoreflect.DoubleKind, protoreflect.FloatKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return nil
	default:
		return fmt.Errorf("field %q is %s, want a numeric scalar", fd.Name(), fd.Kind())
	}
}

// floatOf converts a numeric protoreflect value to a float64.
// It reports false for non-numeric (e.g. bool, string, message) values.
func floatOf(v protoreflect.Value) (float64, bool) {
	switch x := v.Interface().(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}

func (w *worker) faultErr(measured, setPoint float64) *healthpb.HealthCheck_Error {
	return &healthpb.HealthCheck_Error{
		SummaryText: fmt.Sprintf(
			"measured value %.2f deviates %.2f from set point %.2f (tolerance %.2f) for over %s",
			measured, math.Abs(measured-setPoint), setPoint, w.tolerance, w.duration),
	}
}
