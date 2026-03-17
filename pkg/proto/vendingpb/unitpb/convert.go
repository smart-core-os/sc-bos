package unitpb

import (
	"fmt"

	"github.com/smart-core-os/sc-api/go/traits"
)

// Convert converts from one consumable unit to another, if possible.
// Units can only be converted if within the same category, LITER -> CUBIC_METER, but not LITER -> METER.
// Returns an error if conversion is not possible.
func Convert(v float64, from, to traits.Consumable_Unit) (float64, error) {
	if from == to {
		return v, nil
	}

	fromUnit, fromUnitOk := siUnits[from]
	toUnit, toUnitOk := siUnits[to]
	if !fromUnitOk || !toUnitOk || fromUnit.category != toUnit.category {
		return 0, fmt.Errorf("unit: converting %v %v to %v", v, from, to)
	}

	inSI := fromUnit.toSI(v)
	return toUnit.fromSI(inSI), nil
}

// Convert32 is like Convert with type conversion to float32.
func Convert32(v float32, from, to traits.Consumable_Unit) (float32, error) {
	f64, err := Convert(float64(v), from, to)
	return float32(f64), err
}

type category string

const (
	volume category = "volume"
	weight category = "weight"
	length category = "length"
)

type si float64

// Conversions were taken from https://developer.amazon.com/en-US/docs/alexa/device-apis/alexa-property-schemas.html#volume
// Though I assume they are publicly available conversions, I found this source to be easy to consume.

// volume
const (
	liter         si = 1
	milliliter       = liter / 1000
	teaspoon         = 5 * milliliter
	ukGallon         = 4.54609 * liter
	usFluidGallon    = 3.785411784 * liter
	usFluidOunce     = usFluidGallon / 128
	usDryGallon      = 4.40488377086 * liter
	usDryOunce       = usDryGallon / 128

	ukTablespoon    = 3 * teaspoon
	auTablespoon    = 4 * teaspoon
	cubicCentimeter = 1 * milliliter
	cubicMeter      = 1000 * liter
	ukOunce         = ukGallon / 160
	ukQuart         = ukGallon / 4
	ukPint          = ukGallon / 8
	ukCup           = ukGallon / 16
	ukGill          = ukGallon / 32
	ukDram          = ukOunce / 8
	usFluidQuart    = usFluidGallon / 4
	usFluidPint     = usFluidGallon / 8
	usFluidCup      = usFluidGallon / 16
	usTablespoon    = usFluidOunce / 2
	usTeaspoon      = usFluidOunce / 6
	usDram          = usFluidOunce / 8
	usDryQuart      = usDryGallon / 4
	usDryPint       = usDryGallon / 8
	usDryCup        = usDryGallon / 16
	cubicInch       = 16.387064 * milliliter
	cubicFoot       = 28.316846592 * liter
)

// weight
const (
	kilogram si = 1
	gram        = kilogram / 1000
	pound       = 0.45359237 * kilogram
	ounce       = pound / 16

	metricPound = 500 * gram
	milligram   = gram / 1000
	microgram   = milligram / 1000
)

// length/distance
const (
	meter      si = 1
	centimeter    = meter / 100
	millimeter    = meter / 1000
	kilometer     = 1000 * meter
	inch          = 2.54 * centimeter
	foot          = 12 * inch
	yard          = 3 * foot
	mile          = 1760 * yard
	lightyear     = 9460730472580800 * meter
)

type conv struct {
	factor   si
	category category
}

func (c conv) toSI(v float64) float64 {
	return v * float64(c.factor)
}

func (c conv) fromSI(v float64) float64 {
	return v / float64(c.factor)
}

var siUnits = map[traits.Consumable_Unit]conv{
	traits.Consumable_METER:       {meter, length},
	traits.Consumable_LITER:       {liter, volume},
	traits.Consumable_CUBIC_METER: {cubicMeter, volume},
	traits.Consumable_CUP:         {usFluidCup, volume},
	traits.Consumable_KILOGRAM:    {kilogram, weight},
}
