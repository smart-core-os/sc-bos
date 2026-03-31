package job

import (
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func Test_shouldExecuteImmediately(t *testing.T) {
	var tests = []struct {
		name     string
		schedule *jsontypes.Schedule
		now      time.Time
		previous time.Time
		expected bool
	}{
		{
			name:     "now is behind the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("08:59"),
			previous: makeTime("09:00").Add(-24 * time.Hour),
			expected: false,
		},
		{
			name:     "now is on the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "now is after the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:01"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "now is well behind the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("08:00"),
			previous: makeTime("09:00").Add(-24 * time.Hour),
			expected: false,
		},
		{
			name:     "now is well after the next execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("10:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "execution was skipped",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("08:59"),
			previous: makeTime("09:00").Add(-48 * time.Hour),
			expected: true,
		},
		{
			name:     "execution was skipped twice",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:01"),
			previous: makeTime("09:00").Add(-48 * time.Hour),
			expected: true,
		},
		{
			name:     "execution is at halfway point of interval",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("21:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "execution is past halfway point of interval",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("22:00"),
			previous: makeTime("09:00"),
			expected: false,
		},
		{
			name:     "nil previous execution",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:00"),
			previous: time.Time{},
			expected: true,
		},
		{
			name:     "schedule updated to after last execution time",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("09:00"),
			previous: makeTime("08:00"),
			expected: true,
		},
		{
			name:     "schedule updated to before last execution time",
			schedule: jsontypes.MustParseSchedule("0 9 * * *"),
			now:      makeTime("10:00"),
			previous: makeTime("10:00"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExecuteImmediately(tt.schedule, tt.now, tt.previous)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func makeTime(s string) time.Time {
	t, err := time.Parse("15:04", s)
	if err != nil {
		panic(err)
	}
	t = t.Add(time.Hour * 24 * 365) // increment by one year to avoid zero-time issues
	return t
}
