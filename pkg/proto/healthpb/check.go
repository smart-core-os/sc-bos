package healthpb

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// checkBase provides common functionality for health checks of different types.
type checkBase struct {
	check *HealthCheck // nil when disposed
	lifecycle
}

// lifecycle models the transitions a check can go through.
// This type is split out to localise documentation.
type lifecycle struct {
	// When set, onCommit is called after storing a new HealthCheck during write.
	onCommit func(c *HealthCheck)
	// When set, onDispose is called when Dispose is called for the first time.
	onDispose func(c *HealthCheck)
}

// write commits changes made by f as an atomic update.
func (cb *checkBase) write(f func(dst *HealthCheck)) {
	if cb.check == nil {
		return // disposed
	}
	dst := proto.Clone(cb.check).(*HealthCheck)
	f(dst)
	if proto.Equal(cb.check, dst) {
		return
	}
	// apply side effects
	cb.check = dst
	if cb.onCommit != nil {
		cb.onCommit(dst)
	}
}

func makeReliable(dst *HealthCheck) {
	r := dst.GetReliability()
	if r == nil {
		r = &HealthCheck_Reliability{}
		dst.Reliability = r
	}
	oldState := r.GetState()
	r.State = HealthCheck_Reliability_RELIABLE
	r.Cause = nil
	r.Effects = nil
	if oldState != HealthCheck_Reliability_RELIABLE {
		r.ReliableTime = timestamppb.Now()
	}
}

// UpdateMetadata updates the metadata fields of the health check.
// Metadata fields are:
//
//   - DisplayName
//   - Description
//   - OccupantImpact
//   - EquipmentImpact
//   - ComplianceImpacts
//
// Other fields are not updated by this method.
func (cb *checkBase) UpdateMetadata(_ context.Context, c *HealthCheck) {
	cb.write(func(dst *HealthCheck) {
		dst.DisplayName = c.DisplayName
		dst.Description = c.Description
		dst.OccupantImpact = c.OccupantImpact
		dst.EquipmentImpact = c.EquipmentImpact
		dst.ComplianceImpacts = c.ComplianceImpacts
	})

}

// UpdateReliability updates the reliability state of the health check.
// Panics if nr is nil or has an invalid state.
// Reliability timestamps are updated automatically.
// See also [ReliabilityFromErr].
func (cb *checkBase) UpdateReliability(_ context.Context, nr *HealthCheck_Reliability) {
	if nr == nil {
		panic("cannot update reliability to nil")
	}
	if s := nr.GetState(); s == HealthCheck_Reliability_STATE_UNSPECIFIED {
		panic("cannot update reliability to unspecified state")
	}
	if s := nr.GetState(); s == HealthCheck_Reliability_RELIABLE {
		if nr.Cause != nil {
			panic("reliable checks cannot have a cause")
		}
		if nr.Effects != nil {
			panic("reliable checks cannot have effects")
		}
	}

	cb.write(func(dst *HealthCheck) {
		rel := dst.GetReliability()
		if rel == nil {
			rel = &HealthCheck_Reliability{}
			dst.Reliability = rel
		}

		oldState := rel.GetState()
		rel.State = nr.GetState()
		rel.Cause = nr.GetCause()
		rel.Effects = nr.GetEffects()
		// last error stays unless we have a new one
		if v := nr.GetLastError(); v != nil {
			rel.LastError = v
		}
		wasReliable, isReliable := oldState == HealthCheck_Reliability_RELIABLE, rel.GetState() == HealthCheck_Reliability_RELIABLE
		switch {
		case wasReliable == isReliable:
		case wasReliable: // && !isReliable
			rel.UnreliableTime = timestamppb.Now()
		case isReliable: // && !wasReliable
			rel.ReliableTime = timestamppb.Now()
		}
	})
}

// Dispose signals that no more updates to the check will be made.
func (cb *checkBase) Dispose() {
	if cb.check == nil {
		return // already disposed
	}
	c := cb.check
	cb.check = nil
	if cb.onDispose != nil {
		cb.onDispose(c)
	}
}

// ReliabilityFromErr creates a HealthCheck_Reliability from an error.
// If err is nil or is [context.Cancelled], a RELIABLE state is returned.
// If err is [context.DeadlineExceeded], a NO_RESPONSE state is returned.
// gRPC errors are mapped to specific states where possible.
// Other errors are mapped to BAD_RESPONSE.
func ReliabilityFromErr(err error) *HealthCheck_Reliability {
	if err == nil || errors.Is(err, context.Canceled) {
		return &HealthCheck_Reliability{
			State: HealthCheck_Reliability_RELIABLE,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &HealthCheck_Reliability{
			State: HealthCheck_Reliability_NO_RESPONSE,
		}
	}
	if s := grpcErrorToReliabilityState(err); s != 0 {
		e := ErrorToProto(err)
		e.Code = &HealthCheck_Error_Code{Code: status.Code(err).String(), System: "gRPC"}
		return &HealthCheck_Reliability{
			State:     s,
			LastError: e,
		}
	}
	return &HealthCheck_Reliability{
		State:     HealthCheck_Reliability_BAD_RESPONSE,
		LastError: ErrorToProto(err),
	}
}

// grpcErrorToReliabilityState maps gRPC error codes to HealthCheck_Reliability_State values.
func grpcErrorToReliabilityState(err error) HealthCheck_Reliability_State {
	s, ok := status.FromError(err)
	if !ok {
		return 0
	}
	switch s.Code() {
	case codes.NotFound:
		return HealthCheck_Reliability_NOT_FOUND
	case codes.PermissionDenied, codes.Unauthenticated:
		return HealthCheck_Reliability_PERMISSION_DENIED
	case codes.DeadlineExceeded:
		return HealthCheck_Reliability_NO_RESPONSE
	}
	return HealthCheck_Reliability_BAD_RESPONSE
}

// ErrorToProto converts a Go error to a HealthCheck_Error proto.
// If err is nil, nil is returned.
func ErrorToProto(err error) *HealthCheck_Error {
	if err == nil {
		return nil
	}
	return &HealthCheck_Error{SummaryText: err.Error()}
}

func updateStateTimes(c *HealthCheck, oldState, newState HealthCheck_Normality) {
	wasNormal, isNormal := oldState == HealthCheck_NORMAL, newState == HealthCheck_NORMAL
	switch {
	case wasNormal == isNormal:
		// state change has no side effects
	case wasNormal: // && !isNormal
		c.AbnormalTime = timestamppb.Now()
	case isNormal: // && !wasNormal
		c.NormalTime = timestamppb.Now()
	}
}
