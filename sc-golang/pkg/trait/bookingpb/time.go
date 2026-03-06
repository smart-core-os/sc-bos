package bookingpb

import "google.golang.org/protobuf/types/known/timestamppb"

// serverTimestamp returns a timestamppb.Now() but is a var so it can be overridden for tests
var serverTimestamp = func() *timestamppb.Timestamp {
	return timestamppb.Now()
}
