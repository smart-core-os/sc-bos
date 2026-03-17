package time

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-api/go/types/time"
)

func ts(s int64) *timestamppb.Timestamp {
	return &timestamppb.Timestamp{Seconds: s}
}

func between(s1, s2 int64) *time.Period {
	return PeriodBetween(ts(s1), ts(s2))
}

func before(s int64) *time.Period {
	return PeriodBefore(ts(s))
}

func onOrAfter(s int64) *time.Period {
	return PeriodOnOrAfter(ts(s))
}
