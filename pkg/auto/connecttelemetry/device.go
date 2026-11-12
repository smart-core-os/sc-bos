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
	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
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
	// state is each collector's, by index. Its fetch and send state is carried across a device
	// refresh by carryFetchState.
	state []collectorState
}

// collectorState is what a device keeps about one of its collectors between cycles.
type collectorState struct {
	// heartbeat is how long the collector's unchanged points may go unsent, see
	// config.Root.SendOnChange. Zero sends them every cycle. Set when the collector is added.
	heartbeat time.Duration
	// failing is whether the collector failed its last fetch, so a device that can't be read logs
	// once when it starts failing and once when it recovers rather than on every cycle. A dark
	// meter is a state worth one line, not twelve an hour.
	failing bool
	// sent and sentAt are the points last published for an on-change collector, and the
	// scheduled time of the cycle that published them. sentAt is zero until they are sent, and
	// again after a failed fetch, so a collector that recovers sends at once.
	sent   udmi.PointsEvent
	sentAt time.Time
}

// due reports whether points, read at the cycle scheduled for cycle, should be sent.
func (s collectorState) due(points udmi.PointsEvent, cycle time.Time) bool {
	if s.heartbeat == 0 || s.sentAt.IsZero() {
		return true
	}
	return cycle.Sub(s.sentAt) >= s.heartbeat || !points.Equal(s.sent)
}

// sentPoints is the points one collector, by index, contributed to a pointset.
type sentPoints struct {
	index  int
	points udmi.PointsEvent
}

// states returns d.state, one for each collector.
func (d *device) states() []collectorState {
	if len(d.state) != len(d.collectors) {
		d.state = make([]collectorState, len(d.collectors)) // a device built without addCollector
	}
	return d.state
}

// addCollector adds c to d, to be sent every cycle or, with a non-zero heartbeat, on change.
func (d *device) addCollector(c traitCollector, heartbeat time.Duration) {
	d.collectors = append(d.collectors, c)
	d.state = append(d.state, collectorState{heartbeat: heartbeat})
}

// buildTelemetry fetches every collector's current points and merges those due into a single
// UDMI pointset event, see collectorState.due. cycle is the scheduled time of the send.
// ok is false when no points are due. The event timestamp is the most recent reading instant
// across the collectors included, which sent lists for recordSent.
func (d *device) buildTelemetry(ctx context.Context, now, cycle time.Time, logger *zap.Logger) (_ udmi.PointsetEvent, sent []sentPoints, ok bool) {
	merged := udmi.PointsEvent{}
	var ts time.Time
	states := d.states()
	for i, c := range d.collectors {
		st := &states[i]
		fields := []zap.Field{zap.String("device", d.name)}
		if cc, ok := c.(childCollector); ok {
			fields = append(fields, zap.String("child", cc.child))
		}
		points, readingTime, err := c.points(ctx, now)
		if err != nil {
			if st.failing {
				logger.Debug("still failing to fetch trait telemetry", append(fields, zap.Error(err))...)
			} else {
				logger.Warn("failed to fetch trait telemetry, will keep trying", append(fields, zap.Error(err))...)
			}
			st.failing = true
			st.sent, st.sentAt = nil, time.Time{}
			continue
		}
		if st.failing {
			logger.Info("fetching trait telemetry again", fields...)
			st.failing = false
		}
		if !st.due(points, cycle) {
			continue
		}
		sent = append(sent, sentPoints{index: i, points: points})
		for name, v := range points {
			merged[name] = v
		}
		if readingTime.After(ts) {
			ts = readingTime
		}
	}
	if len(merged) == 0 {
		return udmi.PointsetEvent{}, nil, false
	}
	if ts.IsZero() {
		ts = now
	}
	return udmi.PointsetEvent{Timestamp: ts, Version: udmi.PointsetVersion, Points: merged}, sent, true
}

// recordSent notes that sent, from buildTelemetry, was published in the cycle scheduled for cycle,
// so the on-change collectors among them wait for a change or their heartbeat.
func (d *device) recordSent(sent []sentPoints, cycle time.Time) {
	for _, s := range sent {
		st := &d.state[s.index]
		if st.heartbeat == 0 {
			continue // sent every cycle anyway
		}
		st.sent, st.sentAt = s.points, cycle
	}
}

// carryFetchState copies each device's fetch failures and on-change send state from before a
// refresh, for a device that still has the same collectors, so a refresh neither repeats the
// warning for every device that was already failing nor resends every unchanged on-change reading.
// A grouped device's children are added in name order, so its state lines up while its children
// stay the same. Each collector keeps the heartbeat it was added with.
func carryFetchState(from, to map[string]*device) {
	for name, dev := range to {
		old, ok := from[name]
		if !ok || len(old.collectors) != len(dev.collectors) {
			continue
		}
		oldStates, states := old.states(), dev.states()
		for i, o := range oldStates {
			states[i].failing, states[i].sent, states[i].sentAt = o.failing, o.sent, o.sentAt
		}
	}
}

// childCollector is the collector of a child device sent as a point of its parent, see groupTarget.
type childCollector struct {
	traitCollector
	child string // the child's device name, for logs
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

// sensorCollector exports one sensor trait (AirTemperature, AirQualitySensor,
// OccupancySensor, Temperature, Light or Transport) of a device, whose reading is a T, and
// builds discovery from the last reading, so it declares only the points the
// sensor populates, plus any in declared. newSensorCollector seeds last.
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
	// declared are points discovery declares whatever the last reading holds. A Light
	// sends nothing while its level is unknown, and a driver that reports confidence
	// may start every lamp that way until it reports. Discovery goes only every
	// metadataInterval sends, so without this such a lamp could go hours after a
	// restart before Connect learns its unit.
	declared []envPoint
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

// alwaysDeclare adds points to those discovery declares even when the reading doesn't hold
// them, see declared.
func (c *sensorCollector[T]) alwaysDeclare(points ...envPoint) *sensorCollector[T] {
	c.declared = append(c.declared, points...)
	return c
}

func (c *sensorCollector[T]) inventory() map[string]udmi.MetadataPoint {
	inv := envInventory(c.last, c.naming, c.forEach)
	for _, p := range c.declared {
		inv[p.name(c.naming)] = udmi.MetadataPoint{Units: p.units}
	}
	return inv
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

func getTemperature(client temperaturepb.TemperatureApiClient) func(context.Context, string) (*temperaturepb.Temperature, error) {
	return func(ctx context.Context, name string) (*temperaturepb.Temperature, error) {
		return client.GetTemperature(ctx, &temperaturepb.GetTemperatureRequest{Name: name})
	}
}

func getBrightness(client lightpb.LightApiClient) func(context.Context, string) (*lightpb.Brightness, error) {
	return func(ctx context.Context, name string) (*lightpb.Brightness, error) {
		return client.GetBrightness(ctx, &lightpb.GetBrightnessRequest{Name: name})
	}
}

func getTransport(client transportpb.TransportApiClient) func(context.Context, string) (*transportpb.Transport, error) {
	return func(ctx context.Context, name string) (*transportpb.Transport, error) {
		return client.GetTransport(ctx, &transportpb.GetTransportRequest{Name: name})
	}
}

func getOccupancy(client occupancysensorpb.OccupancySensorApiClient) func(context.Context, string) (*occupancysensorpb.Occupancy, error) {
	return func(ctx context.Context, name string) (*occupancysensorpb.Occupancy, error) {
		return client.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: name})
	}
}
