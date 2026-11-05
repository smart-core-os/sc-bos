package connecttelemetry

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
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
	// discoveryOwed is set when the device is found and every metadataInterval cycles,
	// and cleared once its discovery is published, so a failed send is retried next cycle.
	// Only one goroutine touches a device at a time: the loop between cycles, or its
	// publishDevice during one.
	discoveryOwed bool
	// failing is whether each collector, by index, failed its last fetch, so a device that
	// can't be read logs once when it starts failing and once when it recovers rather than on
	// every cycle. A dark meter is a state worth one line, not twelve an hour. Carried across
	// a device refresh by carryFetchState.
	failing []bool
}

// buildTelemetry fetches every collector's current points and merges them into a
// single UDMI pointset event. ok is false when no points could be collected. The
// event timestamp is the most recent reading instant across collectors.
func (d *device) buildTelemetry(ctx context.Context, now time.Time, logger *zap.Logger) (udmi.PointsetEvent, bool) {
	merged := udmi.PointsEvent{}
	var ts time.Time
	if len(d.failing) != len(d.collectors) {
		d.failing = make([]bool, len(d.collectors))
	}
	for i, c := range d.collectors {
		points, readingTime, err := c.points(ctx, now)
		if err != nil {
			if d.failing[i] {
				logger.Debug("still failing to fetch trait telemetry", zap.String("device", d.name), zap.Error(err))
			} else {
				logger.Warn("failed to fetch trait telemetry, will keep trying", zap.String("device", d.name), zap.Error(err))
			}
			d.failing[i] = true
			continue
		}
		if d.failing[i] {
			logger.Info("fetching trait telemetry again", zap.String("device", d.name))
			d.failing[i] = false
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

// carryFetchState copies each device's fetch failures from before a refresh, for a device that
// still has the same collectors, so a refresh does not repeat the warning for every device
// that was already failing.
func carryFetchState(from, to map[string]*device) {
	for name, dev := range to {
		if old, ok := from[name]; ok && len(old.collectors) == len(dev.collectors) {
			dev.failing = old.failing
		}
	}
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
// node as host resource points. Each reading is stamped with the time the
// resourceuse system last sampled it, so a stalled sampler shows as stale rather
// than fresh. last is the most recent reading; inventory builds from it so
// discovery declares the current set of disks. Telemetry is fetched before
// discovery in each cycle, and newCollector seeds last with an initial Get.
// Cycles run one after another, so last needs no lock.
type resourceUseCollector struct {
	name   string
	client resourceusepb.ResourceUseApiClient
	last   *resourceusepb.ResourceUse
}

func (c *resourceUseCollector) points(ctx context.Context, now time.Time) (udmi.PointsEvent, time.Time, error) {
	// ResourceUse carries no timestamp, but Pull seeds with the model's change
	// time: read that first message and close the stream.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := c.client.PullResourceUse(ctx, &resourceusepb.PullResourceUseRequest{Name: c.name})
	if err != nil {
		return nil, time.Time{}, err
	}
	resp, err := stream.Recv()
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(resp.GetChanges()) == 0 {
		return nil, time.Time{}, errors.New("pull resource use returned no changes")
	}
	change := resp.GetChanges()[0]
	c.last = change.GetResourceUse()
	ts := now
	if change.GetChangeTime() != nil {
		ts = change.GetChangeTime().AsTime()
	}
	return resourceUseTelemetry(c.last), ts, nil
}

func (c *resourceUseCollector) inventory() map[string]udmi.MetadataPoint {
	return resourceUseInventory(c.last)
}

// sensorCollector exports one environmental sensor trait (AirTemperature,
// AirQualitySensor or OccupancySensor) of a device, whose reading is a T, and
// builds discovery from the last reading, so it declares only the points the
// sensor populates. newSensorCollector seeds last.
//
// The reading comes from Get, which every driver serves (some poll the device
// on it), and is stamped with the time it was fetched, as a meter's is: a value
// the driver can still read is current. Using the model's change time instead
// would make a sensor whose value doesn't change, such as an empty room's PIR,
// look dead, and land one row per change.
type sensorCollector[T any] struct {
	name    string
	naming  dbo.Naming
	get     func(ctx context.Context, name string) (T, error)
	forEach func(T, dbo.Naming, envPointFunc)
	last    T
}

// newSensorCollector builds a sensorCollector and seeds it with a Get, so
// discovery has an inventory even if the first telemetry fetch fails.
func newSensorCollector[T any](ctx context.Context, name string, naming dbo.Naming,
	get func(context.Context, string) (T, error),
	forEach func(T, dbo.Naming, envPointFunc),
	logger *zap.Logger,
) *sensorCollector[T] {
	c := &sensorCollector[T]{name: name, naming: naming, get: get, forEach: forEach}
	if r, err := get(ctx, name); err != nil {
		logger.Warn("failed to get sensor reading", zap.String("device", name), zap.Error(err))
	} else {
		c.last = r
	}
	return c
}

func (c *sensorCollector[T]) points(ctx context.Context, now time.Time) (udmi.PointsEvent, time.Time, error) {
	r, err := c.get(ctx, c.name)
	if err != nil {
		return nil, time.Time{}, err
	}
	c.last = r
	return envTelemetry(r, c.naming, c.forEach), now, nil
}

func (c *sensorCollector[T]) inventory() map[string]udmi.MetadataPoint {
	return envInventory(c.last, c.naming, c.forEach)
}

func getAirTemperature(client airtemperaturepb.AirTemperatureApiClient) func(context.Context, string) (*airtemperaturepb.AirTemperature, error) {
	return func(ctx context.Context, name string) (*airtemperaturepb.AirTemperature, error) {
		return client.GetAirTemperature(ctx, &airtemperaturepb.GetAirTemperatureRequest{Name: name})
	}
}

func getAirQuality(client airqualitysensorpb.AirQualitySensorApiClient) func(context.Context, string) (*airqualitysensorpb.AirQuality, error) {
	return func(ctx context.Context, name string) (*airqualitysensorpb.AirQuality, error) {
		return client.GetAirQuality(ctx, &airqualitysensorpb.GetAirQualityRequest{Name: name})
	}
}

func getOccupancy(client occupancysensorpb.OccupancySensorApiClient) func(context.Context, string) (*occupancysensorpb.Occupancy, error) {
	return func(ctx context.Context, name string) (*occupancysensorpb.Occupancy, error) {
		return client.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: name})
	}
}
