package time

import "google.golang.org/protobuf/types/known/timestamppb"

// CompareAscending returns -1 if t1 is before t2, 1 if t1 is after t2 and 0 if t1 is equal to t2
func CompareAscending(t1, t2 *timestamppb.Timestamp) int {
	seconds := t1.Seconds - t2.Seconds
	if seconds != 0 {
		return int(seconds)
	}
	return int(t1.Nanos - t2.Nanos)
}
