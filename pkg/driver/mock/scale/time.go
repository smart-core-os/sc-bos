package scale

import (
	"github.com/smart-core-os/sc-bos/pkg/util/timescale"
)

// Time and TimeFunc alias the shared timescale types so existing callers keep
// working; the day-curve mechanism now lives in pkg/util/timescale.
type (
	Time     = timescale.Time
	TimeFunc = timescale.TimeFunc
)

// NineToFive is a Time scaler that reports higher values during working days/hours.
var NineToFive = Time{
	timescale.WorkdayRamp(0.1, 1, nil),       // no work at weekends, must be first
	timescale.LinearRampHour(8, 10, 0.1, 1),  // start of day
	timescale.LinearRampHour(12, 13, 1, 0.5), // start of lunch
	timescale.LinearRampHour(13, 14, 0.5, 1), // end of lunch
	timescale.LinearRampHour(16, 18, 1, 0.1), // end of day
}
