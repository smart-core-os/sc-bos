package connecttelemetry

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
)

// traitCollector fetches telemetry and declares the discovery inventory for one
// trait of one device. New traits are supported by adding a collector.
type traitCollector interface {
	// points fetches the current reading and returns its UDMI points plus the
	// reading instant (now when the source reports no timestamp).
	points(ctx context.Context, now time.Time) (udmi.PointsEvent, time.Time, error)
	// inventory returns this trait's declared point inventory (units/writable) for
	// the discovery message.
	inventory() map[string]udmi.MetadataPoint
}

// device is a discovered device to export: its Smart Core metadata (for
// discovery) plus one collector per supported trait it implements.
type device struct {
	name       string // Smart Core device name; also the deviceRef in the topic (parked gap)
	metaData   *metadatapb.Metadata
	collectors []traitCollector
}

// buildTelemetry fetches every collector's current points and merges them into a
// single UDMI pointset event. ok is false when no points could be collected. The
// event timestamp is the most recent reading instant across collectors.
func (d *device) buildTelemetry(ctx context.Context, now time.Time, logger *zap.Logger) (udmi.PointsetEvent, bool) {
	merged := udmi.PointsEvent{}
	var ts time.Time
	for _, c := range d.collectors {
		points, readingTime, err := c.points(ctx, now)
		if err != nil {
			logger.Warn("failed to fetch trait telemetry", zap.String("device", d.name), zap.Error(err))
			continue
		}
		for name, v := range points {
			merged[name] = v
		}
		if readingTime.After(ts) {
			ts = readingTime
		}
	}
	if len(merged) == 0 {
		return udmi.PointsetEvent{}, false
	}
	if ts.IsZero() {
		ts = now
	}
	return udmi.PointsetEvent{Timestamp: ts, Version: udmi.PointsetVersion, Points: merged}, true
}

// buildDiscovery builds the UDMI device-metadata (discovery) event, merging the
// declared point inventory across the device's collectors.
func (d *device) buildDiscovery(now time.Time) udmi.MetadataEvent {
	inv := map[string]udmi.MetadataPoint{}
	for _, c := range d.collectors {
		for name, p := range c.inventory() {
			inv[name] = p
		}
	}
	return discoveryEvent(d.metaData, inv, now)
}

// meterCollector exports the smartcore.bos.Meter trait as UDMI usage/produced
// points. support (fetched once at discovery) carries the units and production
// capability; it may be nil if the info API is unavailable. naming selects raw vs
// DBO point keys.
type meterCollector struct {
	name    string
	client  meterpb.MeterApiClient
	support *meterpb.MeterReadingSupport
	naming  dbo.Naming
}

func (c *meterCollector) points(ctx context.Context, now time.Time) (udmi.PointsEvent, time.Time, error) {
	r, err := c.client.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: c.name})
	if err != nil {
		return nil, time.Time{}, err
	}
	ev := meterTelemetry(r, c.support, now, c.naming)
	return ev.Points, ev.Timestamp, nil
}

func (c *meterCollector) inventory() map[string]udmi.MetadataPoint {
	return meterInventory(c.support, c.naming)
}

// resourceUseCollector exports the smartcore.bos.ResourceUse trait of the local
// node as host resource points. last is the most recent reading; inventory builds
// from it so discovery declares the current set of disks. Telemetry is fetched
// before discovery in each cycle, and newCollector seeds last with an initial Get.
// Cycles run one after another, so last needs no lock.
type resourceUseCollector struct {
	name   string
	client resourceusepb.ResourceUseApiClient
	last   *resourceusepb.ResourceUse
}

func (c *resourceUseCollector) points(ctx context.Context, now time.Time) (udmi.PointsEvent, time.Time, error) {
	r, err := c.client.GetResourceUse(ctx, &resourceusepb.GetResourceUseRequest{Name: c.name})
	if err != nil {
		return nil, time.Time{}, err
	}
	c.last = r
	return resourceUseTelemetry(r), now, nil
}

func (c *resourceUseCollector) inventory() map[string]udmi.MetadataPoint {
	return resourceUseInventory(c.last)
}
