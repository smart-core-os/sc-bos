package hikcentral

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/api"
)

func TestANPRController_getMostRelevantAppointment(t *testing.T) {
	type parameters struct {
		now          time.Time
		appointments []*api.Appointment
		want         *api.Appointment
	}

	tests := map[string]parameters{
		"one appointment": {
			now: time.Date(2025, 6, 18, 12, 0, 0, 0, time.UTC),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:00")),
				AppointEndTime:   formatTime(makeDateTime("13:00")),
			},
		},
		"2 appointments - ignore end time": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:30")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("10:30")),
					AppointEndTime:   formatTime(makeDateTime("13:30")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:00")),
				AppointEndTime:   formatTime(makeDateTime("12:30")),
			},
		},
		"2 appointments - closer start time": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("11:30")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:30")),
				AppointEndTime:   formatTime(makeDateTime("13:00")),
			},
		},
		"2 appointments - further end time": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("14:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:00")),
				AppointEndTime:   formatTime(makeDateTime("14:00")),
			},
		},
		"2 appointments - pick longest appointment": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:30")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("11:30")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:30")),
				AppointEndTime:   formatTime(makeDateTime("13:00")),
			},
		},
		"2 appointments - end times are irrelevant": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:30")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("11:30")),
					AppointEndTime:   formatTime(makeDateTime("12:15")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:30")),
				AppointEndTime:   formatTime(makeDateTime("12:15")),
			},
		},
		"3 appointments - pick longest appointment": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:30")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("11:30")),
					AppointEndTime:   formatTime(makeDateTime("12:30")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:00")),
				AppointEndTime:   formatTime(makeDateTime("13:00")),
			},
		},
		"now before all appointments": {
			now: makeDateTime("10:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:00")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("12:00")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:00")),
				AppointEndTime:   formatTime(makeDateTime("12:00")),
			},
		},
		"now after all appointments": {
			now: makeDateTime("14:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:00")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("12:00")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("12:00")),
				AppointEndTime:   formatTime(makeDateTime("13:00")),
			},
		},
		"no appointments": {
			now:          makeDateTime("12:00"),
			appointments: []*api.Appointment{},
			want:         nil, // No relevant appointment
		},
		"now between appointments": {
			now: makeDateTime("12:30"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:00")),
				},
				{
					AppointStartTime: formatTime(makeDateTime("13:00")),
					AppointEndTime:   formatTime(makeDateTime("14:30")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("13:00")),
				AppointEndTime:   formatTime(makeDateTime("14:30")),
			},
		},
		"now at the start of an appointment": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("12:00")),
					AppointEndTime:   formatTime(makeDateTime("13:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("12:00")),
				AppointEndTime:   formatTime(makeDateTime("13:00")),
			},
		},
		"now at the end of an appointment": {
			now: makeDateTime("12:00"),
			appointments: []*api.Appointment{
				{
					AppointStartTime: formatTime(makeDateTime("11:00")),
					AppointEndTime:   formatTime(makeDateTime("12:00")),
				},
			},
			want: &api.Appointment{
				AppointStartTime: formatTime(makeDateTime("11:00")),
				AppointEndTime:   formatTime(makeDateTime("12:00")),
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			ret := getMostRelevantAppointment(params.now, params.appointments)

			if diff := cmp.Diff(params.want, ret); diff != "" {
				t.Errorf("getMostRelevantAppointment() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestANPRController_formatPlateNo(t *testing.T) {
	tests := map[string]struct {
		plateNo  string
		expected string
	}{
		"empty plate": {
			plateNo:  "",
			expected: "",
		},
		"valid plate": {
			plateNo:  "ABC123",
			expected: "ABC123",
		},
		"plate with spaces": {
			plateNo:  "A B C 1 2 3",
			expected: "ABC123",
		},
		"plate with special characters": {
			plateNo:  "A-B_C@123",
			expected: "ABC123",
		},
		"plate with mixed case": {
			plateNo:  "aBc123",
			expected: "ABC123",
		},
		"plate with leading/trailing spaces": {
			plateNo:  "  ABC123  ",
			expected: "ABC123",
		},
		"plate with multiple internal spaces": {
			plateNo:  "A  B  C  1  2  3",
			expected: "ABC123",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatPlateNo(test.plateNo)
			if result != test.expected {
				t.Errorf("want %s, got %s", test.expected, result)
			}
		})
	}
}

// this doesn't handle errors well for brevity, please use me in tests only and correctly.
func makeDateTime(t string) time.Time {
	fields := strings.Split(t, ":")
	hour, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		panic(err)
	}
	minute, err := strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		panic(err)
	}
	return time.Date(2025, 6, 18, int(hour), int(minute), 0, 0, time.UTC)
}
