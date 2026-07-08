package sccexporter

import (
	"strings"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// UDMI event subfolders, the trailing topic segment Connect classifies on.
const (
	subfolderPointset  = "pointset"
	subfolderDiscovery = "discovery"
)

// Meter point names. On the trait-poll path there are no raw vendor point names,
// so the meter trait's semantic fields become the UDMI point names.
const (
	meterPointUsage    = "usage"
	meterPointProduced = "produced"
)

// eventTopic builds the UDMI event topic for a device under the Connect ingest
// grammar: <prefix>/devices/<deviceRef>/events/<subfolder>.
//
// NB deviceRef is currently the raw Smart Core device name, which contains '/'.
// That collides with topic segmentation and is a known, deliberately-parked gap
// (see docs/connect-telemetry-ingest.md) — do not rely on the segment count here.
func eventTopic(prefix, deviceRef, subfolder string) string {
	return strings.TrimRight(prefix, "/") + "/devices/" + deviceRef + "/events/" + subfolder
}

func pointsetTopic(prefix, deviceRef string) string {
	return eventTopic(prefix, deviceRef, subfolderPointset)
}

func discoveryTopic(prefix, deviceRef string) string {
	return eventTopic(prefix, deviceRef, subfolderDiscovery)
}

// meterTelemetry maps a meter reading to a UDMI pointset event.
//
// The reading instant is MeterReading.end_time, falling back to now when unset.
// usage is always emitted; produced is only emitted when the meter reports
// production — the support declares a producedUnit, or (when support is unknown)
// the produced value is non-zero — to avoid a spurious constant-zero series for
// consumption-only meters.
func meterTelemetry(r *meterpb.MeterReading, support *meterpb.MeterReadingSupport, now time.Time) udmi.PointsetEvent {
	ts := now
	if r.GetEndTime() != nil {
		ts = r.GetEndTime().AsTime()
	}
	points := udmi.PointsEvent{
		meterPointUsage: {PresentValue: r.GetUsage()},
	}
	if meterHasProduced(r, support) {
		points[meterPointProduced] = udmi.PointValue{PresentValue: r.GetProduced()}
	}
	return udmi.PointsetEvent{Timestamp: ts, Version: udmi.PointsetVersion, Points: points}
}

func meterHasProduced(r *meterpb.MeterReading, support *meterpb.MeterReadingSupport) bool {
	if support != nil {
		return support.GetProducedUnit() != ""
	}
	return r.GetProduced() != 0
}

// meterInventory contributes the meter's declared points to discovery. Meters are
// read-only, so no point is marked writable. Units come from MeterReadingSupport;
// produced is only listed when the meter declares a producedUnit.
func meterInventory(support *meterpb.MeterReadingSupport) map[string]udmi.MetadataPoint {
	inv := map[string]udmi.MetadataPoint{
		meterPointUsage: {Units: support.GetUsageUnit()},
	}
	if support.GetProducedUnit() != "" {
		inv[meterPointProduced] = udmi.MetadataPoint{Units: support.GetProducedUnit()}
	}
	return inv
}

// discoveryEvent builds the UDMI device-metadata (discovery) message from the
// device's Smart Core metadata and the declared point inventory.
//
// system.name prefers the appearance title (falling back to the device name),
// tags carry the building subsystem selector, and location carries the floor.
// site is intentionally left empty here: on the Connect path org/site identity is
// supplied by broker enrichment, not by the payload.
func discoveryEvent(meta *metadatapb.Metadata, inventory map[string]udmi.MetadataPoint, now time.Time) udmi.MetadataEvent {
	ev := udmi.MetadataEvent{
		Timestamp: now,
		Version:   udmi.PointsetVersion,
	}
	if meta != nil {
		if a := meta.GetAppearance(); a != nil {
			ev.System.Name = a.GetTitle()
			ev.System.Description = a.GetDescription()
		}
		if ev.System.Name == "" {
			ev.System.Name = meta.GetName()
		}
		if sub := meta.GetMembership().GetSubsystem(); sub != "" {
			ev.System.Tags = append(ev.System.Tags, sub)
		}
		if loc := meta.GetLocation(); loc != nil && loc.GetFloor() != "" {
			ev.System.Location = &udmi.MetadataLocation{Floor: loc.GetFloor()}
		}
	}
	if len(inventory) > 0 {
		ev.Pointset = &udmi.MetadataPointset{Points: inventory}
	}
	return ev
}
