package healthpb

import (
	"cmp"
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/vanti-dev/sc-bos/pkg/gen"
)

// ValidateValueRange checks that a value range is well-formed.
// r must have at least one of low or high set, and they must be consistent with each other.
// If deadband is set, it must be consistent with the bounds.
// An error explaining the problem is returned, or nil if the range is valid.
func ValidateValueRange(r *gen.HealthCheck_ValueRange) error {
	if r == nil {
		return fmt.Errorf("value range is nil")
	}
	if r.GetLow() == nil && r.GetHigh() == nil {
		return fmt.Errorf("need at least low or high bound")
	}
	// low and high must have the same type (if they are both set)
	b := r.GetLow()
	if bb := r.GetHigh(); bb != nil {
		if b != nil && !SameValueType(b, bb) {
			return fmt.Errorf("value oneof for low and high bounds must match")
		}
		b = bb
	}
	// deadband must be consistent with bounds
	if bb := r.GetDeadband(); bb != nil {
		switch b.GetValue().(type) {
		case *gen.HealthCheck_Value_TimestampValue:
			// deadbands should be durations if bounds are timestamps
			if _, durVal := bb.GetValue().(*gen.HealthCheck_Value_DurationValue); !durVal {
				return fmt.Errorf("deadband must be a duration when bounds are timestamps")
			}
		default:
			// otherwise they should be the same type
			if !SameValueType(b, bb) {
				return fmt.Errorf("value oneof for bounds and deadband must match")
			}
		}
	}
	// bounds don't support bool values
	if _, ok := b.GetValue().(*gen.HealthCheck_Value_BoolValue); ok {
		return fmt.Errorf("cannot use bool values with range bounds")
	}
	return nil
}

// checkRangeState compares v against the normal range r, returning the results as a state.
// current is used during deadband calculations to avoid rapid state changes.
func checkRangeState(r *gen.HealthCheck_ValueRange, v *gen.HealthCheck_Value, current gen.HealthCheck_Normality) gen.HealthCheck_Normality {
	switch current {
	case gen.HealthCheck_LOW:
		if low := r.GetLow(); low != nil && less(v, AddValues(low, r.GetDeadband())) {
			return gen.HealthCheck_LOW
		}
		if high := r.GetHigh(); high != nil && less(high, v) {
			return gen.HealthCheck_HIGH
		}
		return gen.HealthCheck_NORMAL
	case gen.HealthCheck_HIGH:
		if low := r.GetLow(); low != nil && less(v, low) {
			return gen.HealthCheck_LOW
		}
		if high := r.GetHigh(); high != nil && less(high, AddValues(v, r.GetDeadband())) {
			return gen.HealthCheck_HIGH
		}
		return gen.HealthCheck_NORMAL
	default:
		// no deadband processing when transitioning from these states
		if low := r.GetLow(); low != nil && less(v, low) {
			return gen.HealthCheck_LOW
		}
		if high := r.GetHigh(); high != nil && less(high, v) {
			return gen.HealthCheck_HIGH
		}
		return gen.HealthCheck_NORMAL
	}
}

// less returns whether x is less than y, which should have the same underlying type.
func less(x, y *gen.HealthCheck_Value) bool {
	switch v := x.GetValue().(type) {
	case *gen.HealthCheck_Value_StringValue:
		return v.StringValue < y.GetStringValue()
	case *gen.HealthCheck_Value_IntValue:
		return v.IntValue < y.GetIntValue()
	case *gen.HealthCheck_Value_UintValue:
		return v.UintValue < y.GetUintValue()
	case *gen.HealthCheck_Value_FloatValue:
		return v.FloatValue < y.GetFloatValue()
	case *gen.HealthCheck_Value_TimestampValue:
		return v.TimestampValue.AsTime().Before(y.GetTimestampValue().AsTime())
	case *gen.HealthCheck_Value_DurationValue:
		return v.DurationValue.AsDuration() < y.GetDurationValue().AsDuration()
	}
	return false
}

// BoundsCheck updates a health check based on normal bounds and a measured value.
type BoundsCheck struct {
	*checkBase
	checker boundsChecker
}

// newBoundsCheck creates a new bounds check from the given health check definition.
// An error is returned if the checks bounds are inconsistent with itself and the current value.
func newBoundsCheck(c *gen.HealthCheck) (*BoundsCheck, error) {
	r := &BoundsCheck{checkBase: &checkBase{check: c}}
	checker, err := r.prepareChecker(c.GetBounds())
	if err != nil {
		return nil, err
	}
	if checker == nil {
		return nil, fmt.Errorf("bounds are required")
	}
	r.checker = checker
	return r, nil
}

// UpdateBounds updates the bounds and display unit for the health check.
// The new bounds must be valid and consistent with the current value (if any).
func (c *BoundsCheck) UpdateBounds(_ context.Context, b *gen.HealthCheck_Bounds) error {
	checker, err := c.prepareChecker(b)
	if err != nil {
		return err
	}
	c.write(func(dst *gen.HealthCheck) {
		c.checker = checker
		out := dst.GetBounds()
		out.Expected = b.Expected
		out.DisplayUnit = b.DisplayUnit
		c.writeValue(dst, dst.GetBounds().GetCurrentValue()) // recheck the known value against the new bounds
	})
	return nil
}

// UpdateValue updates the check state based on the new value and the current bounds.
func (c *BoundsCheck) UpdateValue(_ context.Context, v *gen.HealthCheck_Value) {
	c.write(func(dst *gen.HealthCheck) {
		c.writeValue(dst, v)
		makeReliable(dst)
	})
}

func (c *BoundsCheck) writeValue(dst *gen.HealthCheck, v *gen.HealthCheck_Value) {
	if c.checker == nil {
		return
	}
	out := dst.GetBounds()
	if out == nil {
		return // shouldn't happen due to checks during creation and UpdateBounds
	}
	oldState := dst.GetNormality()
	newState := c.checker.valueToState(v, dst)
	dst.Normality = newState
	out.CurrentValue = v
	updateStateTimes(dst, oldState, newState)
}

// prepareChecker returns the correct boundsChecker based on the set bounds oneof field.
// An error will be returned if the bounds are invalid.
func (c *BoundsCheck) prepareChecker(b *gen.HealthCheck_Bounds) (boundsChecker, error) {
	switch v := b.GetExpected().(type) {
	case nil:
		return nil, nil // no bounds, no value checking
	case *gen.HealthCheck_Bounds_NormalValue:
		return newNormalValueCheck(b.CurrentValue, v.NormalValue)
	case *gen.HealthCheck_Bounds_AbnormalValue:
		return newAbnormalValueCheck(b.CurrentValue, v.AbnormalValue)
	case *gen.HealthCheck_Bounds_NormalRange:
		return newNormalRangeCheck(b.CurrentValue, v.NormalRange)
	case *gen.HealthCheck_Bounds_NormalValues:
		return newNormalValuesCheck(b.CurrentValue, v.NormalValues.GetValues())
	case *gen.HealthCheck_Bounds_AbnormalValues:
		return newAbnormalValuesCheck(b.CurrentValue, v.AbnormalValues.GetValues())
	}
	return nil, fmt.Errorf("unsupported bounds type %T", b.GetExpected())
}

// boundsChecker compares a value against a normal state, returning the appropriate check state.
type boundsChecker interface {
	valueToState(v *gen.HealthCheck_Value, current *gen.HealthCheck) gen.HealthCheck_Normality
}

func newNormalValueCheck(v, nv *gen.HealthCheck_Value) (*valueCheck, error) {
	// normal value must be consistent with the value (if we have one)
	if v != nil && !SameValueType(v, nv) {
		return nil, fmt.Errorf("normal value oneof must match value type")
	}
	return &valueCheck{value: nv, eq: gen.HealthCheck_NORMAL, neq: gen.HealthCheck_ABNORMAL}, nil
}

func newAbnormalValueCheck(v, av *gen.HealthCheck_Value) (*valueCheck, error) {
	// abnormal value must be consistent with the value (if we have one)
	if v != nil && !SameValueType(v, av) {
		return nil, fmt.Errorf("abnormal value oneof must match value type")
	}
	return &valueCheck{value: av, eq: gen.HealthCheck_ABNORMAL, neq: gen.HealthCheck_NORMAL}, nil
}

// valueCheck checks if a value matches a specific normal value.
type valueCheck struct {
	value   *gen.HealthCheck_Value
	eq, neq gen.HealthCheck_Normality
}

func (c *valueCheck) valueToState(v *gen.HealthCheck_Value, _ *gen.HealthCheck) gen.HealthCheck_Normality {
	if proto.Equal(v, c.value) {
		return c.eq
	}
	return c.neq
}

func newNormalRangeCheck(v *gen.HealthCheck_Value, r *gen.HealthCheck_ValueRange) (*normalRangeCheck, error) {
	if err := ValidateValueRange(r); err != nil {
		return nil, err
	}
	b := cmp.Or(r.GetLow(), r.GetHigh())
	// bounds must be consistent with the value (if we have one)
	if v != nil && !SameValueType(b, v) {
		return nil, fmt.Errorf("bounds oneof must match value type")
	}
	return &normalRangeCheck{bounds: r}, nil
}

// normalRangeCheck checks if a value is within a normal range.
// Unbounded low or high values are supported, but not both.
// Deadband calculations follow the trait spec.
type normalRangeCheck struct {
	bounds *gen.HealthCheck_ValueRange
}

func (c *normalRangeCheck) valueToState(v *gen.HealthCheck_Value, current *gen.HealthCheck) gen.HealthCheck_Normality {
	return checkRangeState(c.bounds, v, current.GetNormality())
}

func newNormalValuesCheck(v *gen.HealthCheck_Value, vs []*gen.HealthCheck_Value) (*valuesCheck, error) {
	if err := validateValuesCheck(v, vs); err != nil {
		return nil, fmt.Errorf("normal %w", err)
	}
	return &valuesCheck{values: vs, in: gen.HealthCheck_NORMAL, nin: gen.HealthCheck_ABNORMAL}, nil
}

func newAbnormalValuesCheck(v *gen.HealthCheck_Value, vs []*gen.HealthCheck_Value) (*valuesCheck, error) {
	if err := validateValuesCheck(v, vs); err != nil {
		return nil, fmt.Errorf("abnormal %w", err)
	}
	return &valuesCheck{values: vs, in: gen.HealthCheck_ABNORMAL, nin: gen.HealthCheck_NORMAL}, nil
}

// valuesCheck checks if a value matches any of a set of normal values.
type valuesCheck struct {
	values  []*gen.HealthCheck_Value
	in, nin gen.HealthCheck_Normality
}

func (c *valuesCheck) valueToState(v *gen.HealthCheck_Value, _ *gen.HealthCheck) gen.HealthCheck_Normality {
	return valuesToState(v, c.values, c.in, c.nin)
}

// validateValuesCheck returns an error if any value in vs is inconsistent with the others, and with v (if set).
func validateValuesCheck(v *gen.HealthCheck_Value, vs []*gen.HealthCheck_Value) error {
	if len(vs) == 0 {
		return fmt.Errorf("values are empty")
	}
	// vs must be consistent with each other
	if !SameValueType(vs...) {
		return fmt.Errorf("values oneof must match")
	}
	// abnormal values must be consistent with the value (if we have one)
	if v != nil {
		if !SameValueType(v, vs[0]) {
			return fmt.Errorf("values oneof must match value type")
		}
	}
	return nil
}

// valuesToState returns eq if v matches any value in vs, otherwise neq.
func valuesToState(v *gen.HealthCheck_Value, vs []*gen.HealthCheck_Value, eq, neq gen.HealthCheck_Normality) gen.HealthCheck_Normality {
	for _, nv := range vs {
		if proto.Equal(v, nv) {
			return eq
		}
	}
	return neq
}
