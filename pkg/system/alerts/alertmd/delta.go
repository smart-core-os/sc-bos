package alertmd

import (
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

func ApplyMdDelta(md *resource.Value, e *alertpb.PullAlertsResponse_Change) error {
	// note: AlertMetadata uses uint32 so we can't use it to store our deltas
	_, err := md.Set(&alertpb.AlertMetadata{}, resource.InterceptBefore(func(old, new proto.Message) {
		oldMd, newMd := old.(*alertpb.AlertMetadata), new.(*alertpb.AlertMetadata)
		proto.Merge(newMd, oldMd)
		// proto.Merge doesn't set maps that are empty but non-nil!
		// Without this explicit copy we'd end up with nil maps that we then assign to, which panics
		if newMd.FloorCounts == nil {
			newMd.FloorCounts = make(map[string]uint32)
		}
		if newMd.ZoneCounts == nil {
			newMd.ZoneCounts = make(map[string]uint32)
		}
		if newMd.SeverityCounts == nil {
			newMd.SeverityCounts = make(map[int32]uint32)
		}
		if newMd.AcknowledgedCounts == nil {
			newMd.AcknowledgedCounts = make(map[bool]uint32)
		}
		if newMd.ResolvedCounts == nil {
			newMd.ResolvedCounts = make(map[bool]uint32)
		}
		if newMd.NeedsAttentionCounts == nil {
			newMd.NeedsAttentionCounts = make(map[string]uint32)
		}
		if newMd.SubsystemCounts == nil {
			newMd.SubsystemCounts = make(map[string]uint32)
		}

		// total
		if e.OldValue == nil && e.NewValue != nil {
			newMd.TotalCount++
		} else if e.OldValue != nil && e.NewValue == nil {
			newMd.TotalCount--
			if newMd.TotalCount < 0 { // shouldn't be needed, but just in case
				newMd.TotalCount = 0
			}
		}

		// ack/nak
		boolMapDelta(e.OldValue, e.NewValue, newMd.AcknowledgedCounts, func(v *alertpb.Alert) bool {
			return v.GetAcknowledgement().GetAcknowledgeTime() != nil
		})
		// resolved/unresolved
		boolMapDelta(e.OldValue, e.NewValue, newMd.ResolvedCounts, func(v *alertpb.Alert) bool {
			return v.GetResolveTime() != nil
		})

		// floors, zones, and severity
		mapDelta(e.GetOldValue().GetFloor(), e.GetNewValue().GetFloor(), newMd.FloorCounts)
		mapDelta(e.GetOldValue().GetZone(), e.GetNewValue().GetZone(), newMd.ZoneCounts)
		mapDelta(int32(e.GetOldValue().GetSeverity()), int32(e.GetNewValue().GetSeverity()), newMd.SeverityCounts)
		mapDelta(e.GetOldValue().GetSubsystem(), e.GetNewValue().GetSubsystem(), newMd.SubsystemCounts)

		// needs attention
		needsAttentionMap(e.GetOldValue(), e.GetNewValue(), newMd.NeedsAttentionCounts)
	}))

	return err
}

func mapDelta[T comparable](o, n T, m map[T]uint32) {
	if o == n {
		return
	}
	var zero T
	if o != zero {
		if c, ok := m[o]; ok && c > 0 {
			m[o] = c - 1
		}
	}
	if n != zero {
		m[n]++
	}
}

func boolMapDelta(o, n *alertpb.Alert, m map[bool]uint32, f func(*alertpb.Alert) bool) {
	switch {
	case o == nil && n != nil:
		m[f(n)]++
	case o != nil && n == nil:
		mapSub(f(o), m)
	case o != nil && n != nil:
		m[f(n)]++
		mapSub(f(o), m)
	}
}

func mapSub[K comparable](k K, m map[K]uint32) {
	if m[k] > 0 {
		m[k]--
	}
}

func mapCmpBool[K comparable](k K, o, n bool, m map[K]uint32) {
	switch {
	case !o && n:
		m[k]++
	case o && !n:
		mapSub(k, m)
	}
}

func needsAttentionMap(o, n *alertpb.Alert, m map[string]uint32) {
	switch {
	case o == nil && n != nil:
		ackResolved, ackUnresolved, nackResolved, nackUnresolved := needsAttentionFlags(n)
		if ackResolved {
			m["ack_resolved"]++
		}
		if ackUnresolved {
			m["ack_unresolved"]++
		}
		if nackResolved {
			m["nack_resolved"]++
		}
		if nackUnresolved {
			m["nack_unresolved"]++
		}
	case o != nil && n == nil:
		ackResolved, ackUnresolved, nackResolved, nackUnresolved := needsAttentionFlags(o)
		if ackResolved {
			mapSub("ack_resolved", m)
		}
		if ackUnresolved {
			mapSub("ack_unresolved", m)
		}
		if nackResolved {
			mapSub("nack_resolved", m)
		}
		if nackUnresolved {
			mapSub("nack_unresolved", m)
		}
	case o != nil && n != nil:
		oAckResolved, oAckUnresolved, oNackResolved, oNackUnresolved := needsAttentionFlags(o)
		nAckResolved, nAckUnresolved, nNackResolved, nNackUnresolved := needsAttentionFlags(n)
		mapCmpBool("ack_resolved", oAckResolved, nAckResolved, m)
		mapCmpBool("ack_unresolved", oAckUnresolved, nAckUnresolved, m)
		mapCmpBool("nack_resolved", oNackResolved, nNackResolved, m)
		mapCmpBool("nack_unresolved", oNackUnresolved, nNackUnresolved, m)
	}
}

func needsAttentionFlags(a *alertpb.Alert) (ackResolved, ackUnresolved, nackResolved, nackUnresolved bool) {
	if a == nil {
		return
	}
	ack := a.GetAcknowledgement().GetAcknowledgeTime() != nil
	resolved := a.GetResolveTime() != nil
	ackResolved = ack && resolved
	ackUnresolved = ack && !resolved
	nackResolved = !ack && resolved
	nackUnresolved = !ack && !resolved
	return
}
