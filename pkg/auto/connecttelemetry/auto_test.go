package connecttelemetry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/connecttelemetry/config"
	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// fakeCollector is a stub traitCollector for exercising device aggregation
// without a live backend.
type fakeCollector struct {
	pts udmi.PointsEvent
	ts  time.Time
	err error
	inv map[string]udmi.MetadataPoint
}

func (f fakeCollector) points(context.Context, time.Time) (udmi.PointsEvent, time.Time, error) {
	return f.pts, f.ts, f.err
}
func (f fakeCollector) inventory() map[string]udmi.MetadataPoint { return f.inv }

func TestDeviceBuildTelemetry(t *testing.T) {
	logger := zap.NewNop()
	now := time.Date(2026, 6, 22, 10, 15, 0, 0, time.UTC)
	earlier := now.Add(-time.Hour)
	later := now.Add(-time.Minute)

	t.Run("merges points and takes the most recent timestamp", func(t *testing.T) {
		dev := &device{name: "m1", collectors: []traitCollector{
			fakeCollector{pts: udmi.PointsEvent{"a": {PresentValue: 1.0}}, ts: earlier},
			fakeCollector{pts: udmi.PointsEvent{"b": {PresentValue: 2.0}}, ts: later},
		}}

		ev, ok := dev.buildTelemetry(context.Background(), now, logger)
		require.True(t, ok)
		assert.Equal(t, later, ev.Timestamp)
		assert.Equal(t, udmi.PointsetVersion, ev.Version)
		assert.Contains(t, ev.Points, "a")
		assert.Contains(t, ev.Points, "b")
	})

	t.Run("skips collectors that error", func(t *testing.T) {
		dev := &device{name: "m1", collectors: []traitCollector{
			fakeCollector{err: errors.New("boom")},
			fakeCollector{pts: udmi.PointsEvent{"b": {PresentValue: 2.0}}, ts: later},
		}}

		ev, ok := dev.buildTelemetry(context.Background(), now, logger)
		require.True(t, ok)
		assert.NotContains(t, ev.Points, "a")
		assert.Contains(t, ev.Points, "b")
	})

	t.Run("no points ⇒ not ok", func(t *testing.T) {
		dev := &device{name: "m1", collectors: []traitCollector{fakeCollector{err: errors.New("boom")}}}
		_, ok := dev.buildTelemetry(context.Background(), now, logger)
		assert.False(t, ok)
	})

	t.Run("falls back to now when collectors report no timestamp", func(t *testing.T) {
		dev := &device{name: "m1", collectors: []traitCollector{
			fakeCollector{pts: udmi.PointsEvent{"a": {PresentValue: 1.0}}}, // zero ts
		}}
		ev, ok := dev.buildTelemetry(context.Background(), now, logger)
		require.True(t, ok)
		assert.Equal(t, now, ev.Timestamp)
	})
}

// unreadableMeter is a meter that cannot currently read its value, as a driver reports it.
type unreadableMeter struct {
	meterpb.UnimplementedMeterApiServer
}

func (unreadableMeter) GetMeterReading(context.Context, *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	return nil, status.Error(codes.Unavailable, "meter usage cannot be read")
}

// TestDeviceBuildTelemetry_unreadableMeter checks a meter that cannot be read publishes no
// pointset, rather than a reading that looks current (SCB-1520).
func TestDeviceBuildTelemetry_unreadableMeter(t *testing.T) {
	client := meterpb.NewMeterApiClient(wrap.ServerToClient(meterpb.MeterApi_ServiceDesc, unreadableMeter{}))
	dev := &device{name: "m1", collectors: []traitCollector{&meterCollector{name: "m1", client: client}}}

	_, ok := dev.buildTelemetry(context.Background(), time.Now(), zap.NewNop())
	assert.False(t, ok)
}

// TestDeviceBuildTelemetry_thinsFailureLogging checks a collector that keeps failing warns once,
// then stays quiet until it recovers, which it says once. At a 5 minute send interval a
// handful of dark meters would otherwise log a dozen warnings an hour each.
func TestDeviceBuildTelemetry_thinsFailureLogging(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	failing := fakeCollector{err: errors.New("unavailable")}
	working := fakeCollector{pts: udmi.PointsEvent{"a": {PresentValue: 1.0}}}
	dev := &device{name: "m1", collectors: []traitCollector{failing}}

	for range 3 {
		dev.buildTelemetry(context.Background(), time.Now(), logger)
	}
	require.Equal(t, 1, logs.FilterLevelExact(zap.WarnLevel).Len(), "one warning however many cycles fail")
	require.Equal(t, 2, logs.FilterLevelExact(zap.DebugLevel).Len())

	dev.collectors[0] = working
	dev.buildTelemetry(context.Background(), time.Now(), logger)
	dev.buildTelemetry(context.Background(), time.Now(), logger)
	require.Equal(t, 1, logs.FilterLevelExact(zap.InfoLevel).Len(), "one line on recovery")

	dev.collectors[0] = failing
	dev.buildTelemetry(context.Background(), time.Now(), logger)
	require.Equal(t, 2, logs.FilterLevelExact(zap.WarnLevel).Len(), "failing again is worth saying again")
}

// TestCarryFetchState checks a device refresh keeps what each device knew about its failing
// collectors, so the refresh every metadata interval does not repeat every warning.
func TestCarryFetchState(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	newDevices := func() map[string]*device {
		return map[string]*device{"m1": {name: "m1", collectors: []traitCollector{fakeCollector{err: errors.New("unavailable")}}}}
	}

	before := newDevices()
	before["m1"].buildTelemetry(context.Background(), time.Now(), logger)
	after := newDevices()
	carryFetchState(before, after)
	after["m1"].buildTelemetry(context.Background(), time.Now(), logger)
	require.Equal(t, 1, logs.Len())

	// a device whose collectors changed starts over, since the indexes no longer line up
	changed := newDevices()
	changed["m1"].collectors = append(changed["m1"].collectors, fakeCollector{})
	carryFetchState(after, changed)
	require.Nil(t, changed["m1"].failing)
}

// TestGetAllTraitImplementorsMeter exercises discovery + collector wiring against
// a live meter model server, then builds telemetry and discovery end-to-end.
func TestGetAllTraitImplementorsMeter(t *testing.T) {
	logger := zap.NewNop()
	root := node.New("meter")

	end := time.Now()
	reading := &meterpb.MeterReading{
		Usage:    123.45,
		Produced: 67.89,
		EndTime:  timestamppb.New(end),
	}
	support := &meterpb.MeterReadingSupport{UsageUnit: "kWh", ProducedUnit: "kWh"}

	devicesApi := devices.NewServer(root)
	meterModel := meterpb.NewModel(resource.WithInitialValue(reading))
	root.Announce("foo",
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterModel))),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{MeterReading: support})),
		node.HasTrait(meterpb.TraitName),
		node.HasServices(root.ClientConn(), devicespb.DevicesApi_ServiceDesc),
	)

	a := &AutoImpl{Services: auto.Services{
		Logger:  logger,
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	allDevices := make(map[string]*device)
	require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, testFetchTimeout, allDevices))

	require.Len(t, allDevices, 1)
	dev := allDevices["foo"]
	require.NotNil(t, dev)
	require.Len(t, dev.collectors, 1)

	// Telemetry: usage + produced, timestamped from the reading's end_time.
	ev, ok := dev.buildTelemetry(context.Background(), time.Now(), logger)
	require.True(t, ok)
	require.Contains(t, ev.Points, dbo.FieldEnergyAccumulator)
	require.Contains(t, ev.Points, dbo.FieldExportedEnergyAccumulator)
	assert.Equal(t, float32(123.45), ev.Points[dbo.FieldEnergyAccumulator].PresentValue)
	assert.Equal(t, float32(67.89), ev.Points[dbo.FieldExportedEnergyAccumulator].PresentValue)
	assert.WithinDuration(t, end, ev.Timestamp, time.Second)

	// Discovery: point inventory carries the units from meter support.
	disc := dev.buildDiscovery(time.Now())
	require.NotNil(t, disc.Pointset)
	assert.Equal(t, "kWh", disc.Pointset.Points[dbo.FieldEnergyAccumulator].Units)
	assert.Equal(t, "kWh", disc.Pointset.Points[dbo.FieldExportedEnergyAccumulator].Units)
}

// TestGetAllTraitImplementorsPaginates confirms every implementor is collected
// when there are more than fit in one page of ListDevices, whether that's the
// server's default (50) or the size we ask for.
func TestGetAllTraitImplementorsPaginates(t *testing.T) {
	const n = 2*listDevicesPageSize + 17
	root := node.New("meters")

	devicesApi := devices.NewServer(root)
	meterModel := meterpb.NewModel()
	support := &meterpb.MeterReadingSupport{UsageUnit: "kWh"}
	for i := range n {
		root.Announce(fmt.Sprintf("meter-%04d", i),
			node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterModel))),
			node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{MeterReading: support})),
			node.HasTrait(meterpb.TraitName),
		)
	}

	a := &AutoImpl{Services: auto.Services{
		Logger:  zap.NewNop(),
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	allDevices := make(map[string]*device)
	require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, testFetchTimeout, allDevices))

	assert.Equal(t, n, len(allDevices))
	assert.NotNil(t, allDevices[fmt.Sprintf("meter-%04d", n-1)], "last device by name")
}

// TestGetAllTraitImplementorsSlowDevices confirms devices that never answer
// collector seeding cost the refresh one fetch timeout between them, not one
// each, and are still exported.
func TestGetAllTraitImplementorsSlowDevices(t *testing.T) {
	const (
		n            = 20
		fetchTimeout = 100 * time.Millisecond
	)
	root := node.New("meters")

	devicesApi := devices.NewServer(root)
	meterModel := meterpb.NewModel()
	for i := range n {
		root.Announce(fmt.Sprintf("hung-%02d", i),
			node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterModel))),
			node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(hungMeterInfo{})),
			node.HasTrait(meterpb.TraitName),
		)
	}

	a := &AutoImpl{Services: auto.Services{
		Logger:  zap.NewNop(),
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	allDevices := make(map[string]*device)
	start := time.Now()
	require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, fetchTimeout, allDevices))
	assert.Less(t, time.Since(start), n*fetchTimeout/2, "seed fetches should time out together")

	assert.Equal(t, n, len(allDevices))
	for name, dev := range allDevices {
		require.Len(t, dev.collectors, 1, name)
		assert.Nil(t, dev.collectors[0].(*meterCollector).support, name)
	}
}

// hungMeterInfo is a MeterInfo server that never answers.
type hungMeterInfo struct {
	meterpb.UnimplementedMeterInfoServer
}

func (hungMeterInfo) DescribeMeterReading(ctx context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// testFetchTimeout bounds device fetches in tests whose devices answer promptly.
const testFetchTimeout = 5 * time.Second

// TestGetAllTraitImplementorsUnsupported confirms an unsupported trait is skipped
// (no collector, no device) rather than erroring.
func TestGetAllTraitImplementorsUnsupported(t *testing.T) {
	logger := zap.NewNop()
	root := node.New("occupancy")

	devicesApi := devices.NewServer(root)
	root.Announce("foo",
		node.HasTrait(trait.OccupancySensor),
		node.HasServices(root.ClientConn(), devicespb.DevicesApi_ServiceDesc),
	)

	a := &AutoImpl{Services: auto.Services{
		Logger:  logger,
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	allDevices := make(map[string]*device)
	require.NoError(t, a.getAllTraitImplementors(context.Background(), trait.OccupancySensor, testFetchTimeout, allDevices))
	assert.Empty(t, allDevices, "unsupported trait should attach no devices")
}

// TestGetAllTraitImplementorsResourceUse confirms ResourceUse is exported for the
// local node only, then builds its telemetry and discovery end-to-end.
func TestGetAllTraitImplementorsResourceUse(t *testing.T) {
	logger := zap.NewNop()
	root := node.New("node1")

	nodeUse := &resourceusepb.ResourceUse{
		Cpu:   &resourceusepb.CpuUse{Utilization: new(float32(12.5))},
		Disks: []*resourceusepb.DiskUse{{MountPoint: "/", Usage: new(uint64(100)), Limit: new(uint64(400)), Utilization: new(float32(25))}},
	}
	cohortUse := &resourceusepb.ResourceUse{Cpu: &resourceusepb.CpuUse{Utilization: new(float32(99))}}

	devicesApi := devices.NewServer(root)
	sampled := time.Date(2026, 10, 12, 8, 59, 0, 0, time.UTC)
	for name, use := range map[string]*resourceusepb.ResourceUse{"node1": nodeUse, "cohort": cohortUse} {
		model := resourceusepb.NewModel(resource.WithInitialValue(use), resource.WithClock(fixedClock(sampled)))
		root.Announce(name,
			node.HasServer(resourceusepb.RegisterResourceUseApiServer, resourceusepb.ResourceUseApiServer(resourceusepb.NewModelServer(model))),
			node.HasTrait(resourceusepb.TraitName),
		)
	}
	root.Announce("node1", node.HasServices(root.ClientConn(), devicespb.DevicesApi_ServiceDesc))

	a := &AutoImpl{Services: auto.Services{
		Logger:  logger,
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	allDevices := make(map[string]*device)
	require.NoError(t, a.getAllTraitImplementors(context.Background(), resourceusepb.TraitName, testFetchTimeout, allDevices))

	require.Len(t, allDevices, 1, "a gateway must not republish its cohort's resources")
	dev := allDevices["node1"]
	require.NotNil(t, dev)
	require.Len(t, dev.collectors, 1)

	// Discovery before any telemetry fetch is served from the seeded reading.
	disc := dev.buildDiscovery(time.Now())
	require.NotNil(t, disc.Pointset)
	assert.Equal(t, udmi.MetadataPoint{Units: "percent"}, disc.Pointset.Points["cpu_percentage_sensor"])
	assert.Equal(t, udmi.MetadataPoint{Units: "bytes", Ref: "/"}, disc.Pointset.Points["disk_datasize_sensor_"+instRoot])

	// Telemetry is stamped with when the node sampled the reading, not when it
	// was fetched, so a stalled sampler arrives stale.
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	ev, ok := dev.buildTelemetry(context.Background(), now, logger)
	require.True(t, ok)
	assert.Equal(t, sampled, ev.Timestamp)
	assert.Equal(t, udmi.PointsEvent{
		"cpu_percentage_sensor":              {PresentValue: float32(12.5)},
		"disk_percentage_sensor_" + instRoot: {PresentValue: float32(25)},
		"disk_datasize_sensor_" + instRoot:   {PresentValue: uint64(100)},
		"disk_datasize_capacity_" + instRoot: {PresentValue: uint64(400)},
	}, ev.Points)
}

// fixedClock is a resource.Clock pinned to one instant.
type fixedClock time.Time

func (c fixedClock) Now() time.Time { return time.Time(c) }

// TestApplyConfigHealthOnlyUnavailable confirms a config whose only output is health
// fails when the node's health checks are unavailable, rather than publishing nothing.
func TestApplyConfigHealthOnlyUnavailable(t *testing.T) {
	a := &AutoImpl{Services: auto.Services{Logger: zap.NewNop()}}
	cfg := config.Root{Health: &config.Health{Enabled: true}}
	err := a.applyConfig(context.Background(), cfg)
	require.ErrorContains(t, err, "nothing to publish")
}

// fakeTelemetrySink records what the telemetry loop publishes.
type fakeTelemetrySink struct {
	mu                sync.Mutex
	upCh              chan struct{} // closed while connected
	failNextDiscovery int           // discovery publishes to fail
	msgs              []telemetryMsg
}

type telemetryMsg struct {
	at    time.Time
	topic string
}

func newFakeTelemetrySink() *fakeTelemetrySink {
	up := make(chan struct{})
	close(up)
	return &fakeTelemetrySink{upCh: up}
}

func (f *fakeTelemetrySink) awaitConnection(ctx context.Context) error {
	f.mu.Lock()
	up := f.upCh
	f.mu.Unlock()
	select {
	case <-up:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// publish fails at once while disconnected, as a *publisher does once its publishTimeout,
// much shorter than a send interval, has passed.
func (f *fakeTelemetrySink) publish(_ context.Context, topic string, _ []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case <-f.upCh:
	default:
		return errors.New("not connected")
	}
	if f.failNextDiscovery > 0 && topic == discoveryTopic("tlm", "foo") {
		f.failNextDiscovery--
		return errors.New("no PUBACK")
	}
	f.msgs = append(f.msgs, telemetryMsg{at: time.Now(), topic: topic})
	return nil
}

func (f *fakeTelemetrySink) setConnected(connected bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !connected {
		f.upCh = make(chan struct{})
		return
	}
	close(f.upCh)
}

func (f *fakeTelemetrySink) setFailNextDiscovery(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNextDiscovery = n
}

// sent returns when each message on topic was published.
func (f *fakeTelemetrySink) sent(topic string) []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	var res []time.Time
	for _, m := range f.msgs {
		if m.topic == topic {
			res = append(res, m.at)
		}
	}
	return res
}

// flakyDevicesClient fails the first failNext ListDevices calls.
type flakyDevicesClient struct {
	devicespb.DevicesApiClient
	mu       sync.Mutex
	failNext int
}

func (c *flakyDevicesClient) ListDevices(ctx context.Context, in *devicespb.ListDevicesRequest, opts ...grpc.CallOption) (*devicespb.ListDevicesResponse, error) {
	c.mu.Lock()
	fail := c.failNext > 0
	if fail {
		c.failNext--
	}
	c.mu.Unlock()
	if fail {
		return nil, errors.New("devices unavailable")
	}
	return c.DevicesApiClient.ListDevices(ctx, in, opts...)
}

type telemetryHarness struct {
	t       *testing.T
	sink    *fakeTelemetrySink
	devices *flakyDevicesClient
	a       *AutoImpl
	start   time.Time
}

// newTelemetryHarness must be called inside a synctest bubble.
// It announces one meter, "foo".
func newTelemetryHarness(t *testing.T) *telemetryHarness {
	root := node.New("meter")
	meterModel := meterpb.NewModel(resource.WithInitialValue(&meterpb.MeterReading{Usage: 123.45}))
	root.Announce("foo",
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterModel))),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{MeterReading: &meterpb.MeterReadingSupport{UsageUnit: "kWh"}})),
		node.HasTrait(meterpb.TraitName),
	)
	devicesClient := &flakyDevicesClient{
		DevicesApiClient: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devices.NewServer(root))),
	}
	a := &AutoImpl{Services: auto.Services{
		Logger:  zaptest.NewLogger(t),
		Node:    root,
		Devices: devicesClient,
	}}
	a.initialiseClients(root)
	return &telemetryHarness{t: t, sink: newFakeTelemetrySink(), devices: devicesClient, a: a}
}

// run starts the telemetry loop, sending every minute and discovery every metadataInterval sends.
func (h *telemetryHarness) run(metadataInterval int) {
	cfg, err := config.ParseConfig(fmt.Appendf(nil, `{
		"traits": [%q],
		"mqtt": {"host": "tls://broker:8883", "clientCertPath": "c", "clientKeyPath": "k",
			"sendInterval": "* * * * *", "metadataInterval": %d}
	}`, meterpb.TraitName, metadataInterval))
	require.NoError(h.t, err)

	ctx, cancel := context.WithCancel(context.Background())
	grp, ctx := errgroup.WithContext(ctx)
	h.a.runTelemetry(ctx, grp, cfg, h.sink)
	h.start = time.Now()
	h.t.Cleanup(func() {
		cancel()
		if err := grp.Wait(); err != nil {
			h.t.Errorf("runTelemetry error = %v", err)
		}
	})
}

// sleepUntil advances the bubble's clock to d after run, then waits for the loop to settle.
func (h *telemetryHarness) sleepUntil(d time.Duration) {
	time.Sleep(time.Until(h.start.Add(d)))
	synctest.Wait()
}

func (h *telemetryHarness) telemetry() []time.Time { return h.sink.sent(pointsetTopic("tlm", "foo")) }
func (h *telemetryHarness) discovery() []time.Time { return h.sink.sent(discoveryTopic("tlm", "foo")) }

// TestRunTelemetry_waitsForConnection is SCB-1512: a broker that refuses the node for part
// of the first interval must not cost that cycle's telemetry and discovery.
func TestRunTelemetry_waitsForConnection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTelemetryHarness(t)
		h.sink.setConnected(false)
		h.run(100)

		h.sleepUntil(90 * time.Second)
		assert.Empty(t, h.telemetry(), "nothing sent while disconnected")
		assert.Empty(t, h.discovery(), "nothing sent while disconnected")

		h.sink.setConnected(true)
		synctest.Wait()
		connected := h.start.Add(90 * time.Second)
		assert.Equal(t, []time.Time{connected}, h.telemetry(), "the first cycle's telemetry, once connected")
		assert.Equal(t, []time.Time{connected}, h.discovery(), "the first cycle's discovery, once connected")

		h.sleepUntil(2*time.Minute + time.Second)
		assert.Equal(t, []time.Time{connected, h.start.Add(2 * time.Minute)}, h.telemetry(), "telemetry at the next send")
		assert.Len(t, h.discovery(), 1, "discovery not again until cycle N")
	})
}

// TestRunTelemetry_skipsCycleWhenConnectionTakesTooLong confirms a cycle that can't reach the
// broker before the next send gives way to it, and the discovery it owed goes then.
func TestRunTelemetry_skipsCycleWhenConnectionTakesTooLong(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTelemetryHarness(t)
		h.sink.setConnected(false)
		h.run(100)

		h.sleepUntil(150 * time.Second)
		assert.Empty(t, h.telemetry(), "nothing sent while disconnected")

		h.sink.setConnected(true)
		synctest.Wait()
		connected := h.start.Add(150 * time.Second)
		assert.Equal(t, []time.Time{connected}, h.telemetry(), "one cycle sent, the skipped one not caught up")
		assert.Equal(t, []time.Time{connected}, h.discovery(), "the owed discovery")

		h.sleepUntil(3*time.Minute + time.Second)
		assert.Len(t, h.telemetry(), 2, "telemetry at the next send")
		assert.Len(t, h.discovery(), 1, "discovery not again until cycle N")
	})
}

// TestRunTelemetry_retriesFailedDiscovery confirms discovery that fails to publish is resent
// on the next cycle, without moving the every-N-cycles cadence.
func TestRunTelemetry_retriesFailedDiscovery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTelemetryHarness(t)
		h.sink.setFailNextDiscovery(1)
		h.run(3)

		h.sleepUntil(time.Minute + time.Second)
		assert.Len(t, h.telemetry(), 1)
		assert.Empty(t, h.discovery(), "first discovery failed")

		h.sleepUntil(2*time.Minute + time.Second)
		assert.Equal(t, []time.Time{h.start.Add(2 * time.Minute)}, h.discovery(), "resent on the next cycle")

		h.sleepUntil(4*time.Minute + time.Second)
		assert.Len(t, h.telemetry(), 4)
		assert.Equal(t, []time.Time{h.start.Add(2 * time.Minute), h.start.Add(4 * time.Minute)}, h.discovery(),
			"not again until cycle N")
	})
}

// TestRunTelemetry_retriesFailedRefresh confirms a failed first device list is retried on the
// next cycle, rather than leaving nothing to export until cycle N.
func TestRunTelemetry_retriesFailedRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTelemetryHarness(t)
		h.devices.failNext = 1
		h.run(100)

		h.sleepUntil(time.Minute + time.Second)
		assert.Empty(t, h.telemetry(), "no devices to export")
		assert.Empty(t, h.discovery(), "no devices to export")

		h.sleepUntil(2*time.Minute + time.Second)
		assert.Equal(t, []time.Time{h.start.Add(2 * time.Minute)}, h.telemetry(), "found on the retried refresh")
		assert.Equal(t, []time.Time{h.start.Add(2 * time.Minute)}, h.discovery(), "found on the retried refresh")
	})
}
