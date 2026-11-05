package connecttelemetry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"maps"
	"slices"
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
	"github.com/smart-core-os/sc-bos/pkg/connect"
	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	zoneoccupancy "github.com/smart-core-os/sc-bos/pkg/zone/feature/occupancy"
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
	require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, testFetchTimeout, false, nil, allDevices))

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

// TestGetAllTraitImplementorsSkipsZones confirms zones are left out unless
// includeZones is set, while meters with no device type are always kept.
func TestGetAllTraitImplementorsSkipsZones(t *testing.T) {
	root := node.New("zones")

	devicesApi := devices.NewServer(root)
	meterModel := meterpb.NewModel()
	support := &meterpb.MeterReadingSupport{UsageUnit: "kWh"}
	announce := func(name string, opts ...node.Feature) {
		root.Announce(name, append([]node.Feature{
			node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterModel))),
			node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{MeterReading: support})),
			node.HasTrait(meterpb.TraitName),
		}, opts...)...)
	}
	announce("meter", node.HasDeviceType(metadatapb.Metadata_DEVICE))
	announce("untyped")
	announce("zone", node.HasDeviceType(metadatapb.Metadata_ZONE))

	a := &AutoImpl{Services: auto.Services{
		Logger:  zap.NewNop(),
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	for _, tc := range []struct {
		includeZones bool
		want         []string
	}{
		{includeZones: false, want: []string{"meter", "untyped"}},
		{includeZones: true, want: []string{"meter", "untyped", "zone"}},
	} {
		allDevices := make(map[string]*device)
		require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, testFetchTimeout, tc.includeZones, nil, allDevices))
		assert.ElementsMatch(t, tc.want, slices.Collect(maps.Keys(allDevices)), "includeZones=%v", tc.includeZones)
	}
}

// TestRefreshDevicesAllowlist checks a trait with a devices entry exports only
// the devices named, admitting a named zone, while other traits are unchanged.
func TestRefreshDevicesAllowlist(t *testing.T) {
	root := node.New("site")
	devicesApi := devices.NewServer(root)
	announceOccupancy := func(name string, deviceType metadatapb.Metadata_DeviceType) {
		model := occupancysensorpb.NewModel(resource.WithInitialValue(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 1}))
		root.Announce(name,
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model))),
			node.HasTrait(trait.OccupancySensor),
			node.HasDeviceType(deviceType),
		)
	}
	announceOccupancy("pir", metadatapb.Metadata_DEVICE)
	announceOccupancy("other-zone", metadatapb.Metadata_ZONE)
	announceOccupancy("treehouse", metadatapb.Metadata_ZONE)
	meterModel := meterpb.NewModel()
	root.Announce("meter",
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(meterpb.NewModelServer(meterModel))),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(&meterpb.InfoServer{MeterReading: &meterpb.MeterReadingSupport{UsageUnit: "kWh"}})),
		node.HasTrait(meterpb.TraitName),
	)

	a := &AutoImpl{Services: auto.Services{
		Logger:  zap.NewNop(),
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	for _, tc := range []struct {
		name        string
		devices     map[string][]string
		want        []string
		wantMissing []string
	}{
		{
			name: "no allowlist",
			want: []string{"pir", "meter"},
		},
		{
			name:    "occupancy allowlist",
			devices: map[string][]string{string(trait.OccupancySensor): {"treehouse"}},
			want:    []string{"treehouse", "meter"},
		},
		{
			name:        "allowlist naming a missing device",
			devices:     map[string][]string{string(trait.OccupancySensor): {"treehouse", "tree-house"}},
			want:        []string{"treehouse", "meter"},
			wantMissing: []string{"tree-house"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zap.WarnLevel)
			a.Logger = zap.New(core)
			cfg := config.Root{
				Traits:       []string{string(trait.OccupancySensor), string(meterpb.TraitName)},
				FetchTimeout: &jsontypes.Duration{Duration: testFetchTimeout},
				Devices:      tc.devices,
			}
			allDevices := make(map[string]*device)
			require.NoError(t, a.refreshDevices(context.Background(), cfg, allDevices))
			assert.ElementsMatch(t, tc.want, slices.Collect(maps.Keys(allDevices)))

			var missing []string
			for _, e := range logs.FilterMessage("configured device not found for trait").All() {
				missing = append(missing, e.ContextMap()["device"].(string))
			}
			assert.ElementsMatch(t, tc.wantMissing, missing)
		})
	}
}

// TestRefreshDevicesEnterLeaveZone runs a zone counting people with an
// enter/leave sensor through the allowlist, checking it sends its count,
// including a count of 0 once everyone has left.
func TestRefreshDevicesEnterLeaveZone(t *testing.T) {
	root := node.New("site")
	devicesApi := devices.NewServer(root)
	counter := enterleavesensorpb.NewModel()
	root.Announce("counter",
		node.HasServer(enterleavesensorpb.RegisterEnterLeaveSensorApiServer, enterleavesensorpb.EnterLeaveSensorApiServer(enterleavesensorpb.NewModelServer(counter))),
		node.HasTrait(trait.EnterLeaveSensor),
	)
	root.Announce("treehouse", node.HasDeviceType(metadatapb.Metadata_ZONE))
	zoneSvc := zoneoccupancy.Feature.New(zone.Services{Logger: zap.NewNop(), Node: root, Devices: &zone.Devices{}})
	_, err := zoneSvc.Configure([]byte(`{"name":"treehouse","enterLeaveOccupancySensors":["counter"]}`))
	require.NoError(t, err)
	_, err = zoneSvc.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = zoneSvc.Stop() })

	a := &AutoImpl{Services: auto.Services{
		Logger:  zap.NewNop(),
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	cfg := config.Root{
		Traits:       []string{string(trait.OccupancySensor)},
		FetchTimeout: &jsontypes.Duration{Duration: testFetchTimeout},
		Devices:      map[string][]string{string(trait.OccupancySensor): {"treehouse"}},
	}
	var allDevices map[string]*device
	// The zone announces once its config is applied, which may land after Start returns.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		allDevices = make(map[string]*device)
		require.NoError(c, a.refreshDevices(context.Background(), cfg, allDevices))
		require.Contains(c, allDevices, "treehouse")
	}, 5*time.Second, 10*time.Millisecond)

	sent := func() udmi.PointsEvent {
		ev, ok := allDevices["treehouse"].buildTelemetry(context.Background(), time.Now(), zap.NewNop())
		require.True(t, ok)
		return ev.Points
	}
	move := func(dir enterleavesensorpb.EnterLeaveEvent_Direction, n int) {
		for range n {
			require.NoError(t, counter.CreateEnterLeaveEvent(&enterleavesensorpb.EnterLeaveEvent{Direction: dir}))
		}
	}

	move(enterleavesensorpb.EnterLeaveEvent_ENTER, 3)
	move(enterleavesensorpb.EnterLeaveEvent_LEAVE, 1)
	assert.Equal(t, udmi.PointsEvent{
		dbo.FieldZoneOccupancyStatus: {PresentValue: dbo.OccupancyOccupied},
		dbo.FieldZoneOccupancyCount:  {PresentValue: int32(2)},
	}, sent())

	move(enterleavesensorpb.EnterLeaveEvent_LEAVE, 2)
	assert.Equal(t, udmi.PointsEvent{
		dbo.FieldZoneOccupancyStatus: {PresentValue: dbo.OccupancyUnoccupied},
		dbo.FieldZoneOccupancyCount:  {PresentValue: int32(0)},
	}, sent(), "a zone counting people sends a true 0")
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
	require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, testFetchTimeout, false, nil, allDevices))

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
	require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, fetchTimeout, false, nil, allDevices))
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
	root := node.New("brightness")

	devicesApi := devices.NewServer(root)
	root.Announce("foo",
		node.HasTrait(trait.BrightnessSensor),
		node.HasServices(root.ClientConn(), devicespb.DevicesApi_ServiceDesc),
	)

	a := &AutoImpl{Services: auto.Services{
		Logger:  logger,
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	allDevices := make(map[string]*device)
	require.NoError(t, a.getAllTraitImplementors(context.Background(), trait.BrightnessSensor, testFetchTimeout, false, nil, allDevices))
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
	require.NoError(t, a.getAllTraitImplementors(context.Background(), resourceusepb.TraitName, testFetchTimeout, false, nil, allDevices))

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

// fakeCredential is a connect.Credential whose state the test sets.
type fakeCredential struct {
	initial connect.State
	changes chan connect.State
}

func newFakeCredential(initial connect.State) *fakeCredential {
	return &fakeCredential{initial: initial, changes: make(chan connect.State)}
}

func (c *fakeCredential) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeCredential) State() connect.State { return c.initial }

func (c *fakeCredential) PullState(ctx context.Context) (connect.State, <-chan connect.State) {
	out := make(chan connect.State)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case st := <-c.changes:
				select {
				case out <- st:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return c.initial, out
}

// set delivers st to the watcher and waits for it to act on it, including for any run it stops to disconnect.
// Call it inside a synctest bubble.
func (c *fakeCredential) set(st connect.State) {
	c.changes <- st
	time.Sleep(time.Second)
	synctest.Wait()
}

// failureCheck is a service.SystemCheck that records the errors it is marked failed with.
type failureCheck struct {
	mu       sync.Mutex
	failures []error
}

func (c *failureCheck) Dispose()     {}
func (c *failureCheck) MarkRunning() {}
func (c *failureCheck) MarkFailed(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = append(c.failures, err)
}

func (c *failureCheck) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.failures)
}

// enrolmentRuns records the runs followEnrolment starts.
type enrolmentRuns struct {
	mu       sync.Mutex
	started  []connect.State
	active   int
	overlaps int // runs started while another was still running
}

func (r *enrolmentRuns) run(ctx context.Context, st connect.State) error {
	r.mu.Lock()
	r.started = append(r.started, st)
	if r.active > 0 {
		r.overlaps++
	}
	r.active++
	r.mu.Unlock()

	<-ctx.Done()
	// a run takes a moment to disconnect, which the next must wait for
	time.Sleep(500 * time.Millisecond)

	r.mu.Lock()
	r.active--
	r.mu.Unlock()
	return nil
}

func (r *enrolmentRuns) state() (started []connect.State, active, overlaps int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]connect.State(nil), r.started...), r.active, r.overlaps
}

func TestFollowEnrolment(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		enrolled := func(c connect.Connectivity, nodeID, endpoint string) connect.State {
			return connect.State{Connectivity: c, NodeID: nodeID, APIEndpoint: endpoint}
		}
		const api = "https://connect.example.com"
		n1 := enrolled(connect.Connecting, "n1", api)
		n2 := enrolled(connect.Connected, "n2", api)
		n3 := enrolled(connect.Connecting, "n3", api)
		n3Moved := enrolled(connect.Connected, "n3", "https://other.example.com")

		cred := newFakeCredential(connect.State{Connectivity: connect.NotEnrolled})
		check := &failureCheck{}
		runs := &enrolmentRuns{}
		ctx, cancel := context.WithCancel(t.Context())
		returned := make(chan error, 1)
		go func() { returned <- followEnrolment(ctx, cred, check, zap.NewNop(), runs.run) }()

		expect := func(wantStarted []connect.State, wantActive int) {
			t.Helper()
			started, active, overlaps := runs.state()
			assert.Equal(t, wantStarted, started)
			assert.Equal(t, wantActive, active, "active runs")
			assert.Zero(t, overlaps, "runs started while another was running")
		}

		synctest.Wait()
		expect(nil, 0)
		require.Equal(t, 1, check.count(), "check not marked failed before enrolment")
		assert.ErrorIs(t, check.failures[0], errNotEnrolled)

		cred.set(n1)
		expect([]connect.State{n1}, 1)

		// check-ins change only the connectivity, which needs no new connection
		cred.set(enrolled(connect.Connected, "n1", api))
		cred.set(enrolled(connect.Failed, "n1", api))
		expect([]connect.State{n1}, 1)

		// re-enrolled under another node id
		cred.set(n2)
		expect([]connect.State{n1, n2}, 1)

		cred.set(connect.State{Connectivity: connect.NotEnrolled})
		expect([]connect.State{n1, n2}, 0)
		require.Equal(t, 2, check.count(), "check not marked failed after unlinking")
		assert.ErrorIs(t, check.failures[1], errNotEnrolled)

		cred.set(n3)
		expect([]connect.State{n1, n2, n3}, 1)

		// re-enrolled with another endpoint
		cred.set(n3Moved)
		expect([]connect.State{n1, n2, n3, n3Moved}, 1)

		cancel()
		time.Sleep(time.Second)
		synctest.Wait()
		assert.ErrorIs(t, <-returned, context.Canceled)
		expect([]connect.State{n1, n2, n3, n3Moved}, 0)
		assert.Equal(t, 2, check.count(), "check marked failed on stop")
	})
}

// TestFollowEnrolment_runFails checks a run that ends by itself, as it does on an error, ends followEnrolment too.
func TestFollowEnrolment_runFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cred := newFakeCredential(connect.State{Connectivity: connect.Connected, NodeID: "n1", APIEndpoint: "https://connect.example.com"})
		runErr := errors.New("boom")
		err := followEnrolment(t.Context(), cred, nil, zap.NewNop(), func(context.Context, connect.State) error {
			return runErr
		})
		assert.ErrorIs(t, err, runErr)
	})
}

// TestRun checks each publish is as the node the credential is enrolled as.
func TestRun(t *testing.T) {
	const api = "https://connect.example.com"
	n1 := connect.State{Connectivity: connect.Connected, NodeID: "n1", APIEndpoint: api}
	n2 := connect.State{Connectivity: connect.Connected, NodeID: "n2", APIEndpoint: api}

	t.Run("cloud credential", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			cred := newFakeCredential(n1)
			a := &AutoImpl{Services: auto.Services{Logger: zap.NewNop(), CloudCredential: cred}}
			var mu sync.Mutex
			var published []string
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			go a.run(ctx, config.Mqtt{UseCloudCredential: true}, func(ctx context.Context, nodeID string) error {
				mu.Lock()
				published = append(published, nodeID)
				mu.Unlock()
				<-ctx.Done()
				return nil
			})
			synctest.Wait()
			cred.set(n2)
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"n1", "n2"}, published)
		})
	})

	t.Run("file path", func(t *testing.T) {
		a := &AutoImpl{Services: auto.Services{Logger: zap.NewNop(), CloudCredential: newFakeCredential(n1)}}
		var published []string
		err := a.run(t.Context(), config.Mqtt{}, func(_ context.Context, nodeID string) error {
			published = append(published, nodeID)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"n1"}, published)
	})
}

// TestGetAllTraitImplementorsEnvironment exercises discovery and collector wiring
// for the environmental sensor traits against live model servers: one sensor
// implementing all three, as a multi-sensor does, publishes one merged pointset.
func TestGetAllTraitImplementorsEnvironment(t *testing.T) {
	logger := zap.NewNop()
	root := node.New("env")

	sampled := time.Date(2026, 11, 5, 8, 55, 0, 0, time.UTC)
	clock := resource.WithClock(fixedClock(sampled))
	airTemperature := airtemperaturepb.NewModel(clock, resource.WithInitialValue(&airtemperaturepb.AirTemperature{
		Mode:               airtemperaturepb.AirTemperature_HEAT,
		AmbientTemperature: &typespb.Temperature{ValueCelsius: 22.5},
		AmbientHumidity:    new(float32(41)),
	}))
	airQuality := airqualitysensorpb.NewModel(clock, resource.WithInitialValue(&airqualitysensorpb.AirQuality{
		CarbonDioxideLevel: new(float32(612)),
		Score:              new(float32(90)),
	}))
	occupancy := occupancysensorpb.NewModel(clock, resource.WithInitialValue(&occupancysensorpb.Occupancy{
		State:           occupancysensorpb.Occupancy_OCCUPIED,
		PeopleCount:     4,
		StateChangeTime: timestamppb.New(sampled.Add(-3 * time.Hour)),
	}))

	devicesApi := devices.NewServer(root)
	root.Announce("sensor",
		node.HasServer(airtemperaturepb.RegisterAirTemperatureApiServer, airtemperaturepb.AirTemperatureApiServer(airtemperaturepb.NewModelServer(airTemperature))),
		node.HasServer(airqualitysensorpb.RegisterAirQualitySensorApiServer, airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(airQuality))),
		node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(occupancy))),
		node.HasTrait(trait.AirTemperature),
		node.HasTrait(trait.AirQualitySensor),
		node.HasTrait(trait.OccupancySensor),
	)
	root.Announce("env", node.HasServices(root.ClientConn(), devicespb.DevicesApi_ServiceDesc))

	a := &AutoImpl{Services: auto.Services{
		Logger:  logger,
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)

	allDevices := make(map[string]*device)
	for _, tn := range []trait.Name{trait.AirTemperature, trait.AirQualitySensor, trait.OccupancySensor} {
		require.NoError(t, a.getAllTraitImplementors(context.Background(), tn, testFetchTimeout, false, nil, allDevices))
	}
	require.Len(t, allDevices, 1)
	dev := allDevices["sensor"]
	require.NotNil(t, dev)
	require.Len(t, dev.collectors, 3)

	// Discovery before any telemetry fetch is served from the seeded readings,
	// with every numeric point's unit.
	disc := dev.buildDiscovery(time.Now())
	require.NotNil(t, disc.Pointset)
	assert.Equal(t, map[string]udmi.MetadataPoint{
		dbo.FieldZoneAirTemperatureSensor:      {Units: dbo.UnitDegreesCelsius},
		dbo.FieldZoneAirRelativeHumiditySensor: {Units: dbo.UnitPercentRelativeHumidity},
		dbo.FieldZoneAirCO2ConcentrationSensor: {Units: dbo.UnitPartsPerMillion},
		dbo.FieldZoneOccupancyStatus:           {},
		dbo.FieldZoneOccupancyCount:            {},
	}, disc.Pointset.Points)

	// Telemetry is stamped with when it was fetched, not when the readings
	// last changed, nor when the room filled: a value still read is current.
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	ev, ok := dev.buildTelemetry(context.Background(), now, logger)
	require.True(t, ok)
	assert.Equal(t, now, ev.Timestamp)
	assert.Equal(t, udmi.PointsEvent{
		dbo.FieldZoneAirTemperatureSensor:      {PresentValue: 22.5},
		dbo.FieldZoneAirRelativeHumiditySensor: {PresentValue: float32(41)},
		dbo.FieldZoneAirCO2ConcentrationSensor: {PresentValue: float32(612)},
		dbo.FieldZoneOccupancyStatus:           {PresentValue: dbo.OccupancyOccupied},
		dbo.FieldZoneOccupancyCount:            {PresentValue: int32(4)},
	}, ev.Points)

	// A reading that gains a field declares it at the next discovery.
	_, err := airQuality.UpdateAirQuality(&airqualitysensorpb.AirQuality{ParticulateMatter_25: new(float32(2.5))},
		resource.WithUpdatePaths("particulate_matter_25"))
	require.NoError(t, err)
	_, ok = dev.buildTelemetry(context.Background(), now, logger)
	require.True(t, ok)
	assert.Equal(t, udmi.MetadataPoint{Units: dbo.UnitMicrogramsPerCubicMeter},
		dev.buildDiscovery(now).Pointset.Points[dbo.FieldZoneAirPM2pt5DensitySensor])
}

// TestSensorCollectorPoints checks a sensor reading comes from Get and is
// stamped with the fetch time, as a meter's is.
func TestSensorCollectorPoints(t *testing.T) {
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)

	t.Run("the reading goes stamped now", func(t *testing.T) {
		c := &sensorCollector[*occupancysensorpb.Occupancy]{
			name: "s",
			get: func(context.Context, string) (*occupancysensorpb.Occupancy, error) {
				return &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(now.Add(-8 * time.Hour))}, nil
			},
			forEach: forEachOccupancyPointFor(countingCounts),
		}
		pts, ts, err := c.points(context.Background(), now)
		require.NoError(t, err)
		assert.Equal(t, now, ts, "an empty room still reads now, not when it emptied")
		assert.Equal(t, udmi.PointsEvent{
			dbo.FieldZoneOccupancyStatus: {PresentValue: dbo.OccupancyUnoccupied},
			dbo.FieldZoneOccupancyCount:  {PresentValue: int32(0)},
		}, pts)
		assert.Equal(t, map[string]udmi.MetadataPoint{
			dbo.FieldZoneOccupancyStatus: {},
			dbo.FieldZoneOccupancyCount:  {},
		}, c.inventory(), "discovery follows the reading sent")
	})

	t.Run("a failed get is a failed fetch", func(t *testing.T) {
		c := &sensorCollector[*occupancysensorpb.Occupancy]{
			name: "s",
			get: func(context.Context, string) (*occupancysensorpb.Occupancy, error) {
				return nil, status.Error(codes.Unavailable, "sensor offline")
			},
			forEach: forEachOccupancyPointFor(countingCounts),
		}
		_, _, err := c.points(context.Background(), now)
		assert.Error(t, err)
	})
}

// occupancyInfo describes an occupancy sensor's people counting.
type occupancyInfo struct {
	occupancysensorpb.UnimplementedOccupancySensorInfoServer
	maxPeople int32
}

func (i occupancyInfo) DescribeOccupancy(context.Context, *occupancysensorpb.DescribeOccupancyRequest) (*occupancysensorpb.OccupancySupport, error) {
	return &occupancysensorpb.OccupancySupport{MaxPeople: i.maxPeople}, nil
}

// TestGetAllTraitImplementorsOccupancyCounting checks the occupancy collector
// sends counts by what each sensor says about its counting, read when the
// device is found.
func TestGetAllTraitImplementorsOccupancyCounting(t *testing.T) {
	root := node.New("occupancy")
	devicesApi := devices.NewServer(root)
	announce := func(name string, reading *occupancysensorpb.Occupancy, info ...node.Feature) {
		model := occupancysensorpb.NewModel(resource.WithInitialValue(reading))
		root.Announce(name, append([]node.Feature{
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model))),
			node.HasTrait(trait.OccupancySensor),
		}, info...)...)
	}
	empty := &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED}
	announce("counter", empty, node.HasServer(occupancysensorpb.RegisterOccupancySensorInfoServer, occupancysensorpb.OccupancySensorInfoServer(occupancyInfo{maxPeople: 40})))
	announce("presence", &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 2}, node.HasServer(occupancysensorpb.RegisterOccupancySensorInfoServer, occupancysensorpb.OccupancySensorInfoServer(occupancyInfo{})))
	announce("pir", empty)

	a := &AutoImpl{Services: auto.Services{
		Logger:  zap.NewNop(),
		Node:    root,
		Devices: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
	}}
	a.initialiseClients(root)
	allDevices := make(map[string]*device)
	require.NoError(t, a.getAllTraitImplementors(context.Background(), trait.OccupancySensor, testFetchTimeout, false, nil, allDevices))
	require.Len(t, allDevices, 3)

	sent := func(name string) udmi.PointsEvent {
		ev, ok := allDevices[name].buildTelemetry(context.Background(), time.Now(), zap.NewNop())
		require.True(t, ok, name)
		return ev.Points
	}
	assert.Equal(t, udmi.PointsEvent{
		dbo.FieldZoneOccupancyStatus: {PresentValue: dbo.OccupancyUnoccupied},
		dbo.FieldZoneOccupancyCount:  {PresentValue: int32(0)},
	}, sent("counter"), "a counter's empty room is a true 0")
	assert.Equal(t, udmi.PointsEvent{
		dbo.FieldZoneOccupancyStatus: {PresentValue: dbo.OccupancyOccupied},
	}, sent("presence"), "a sensor that says it doesn't count sends none")
	assert.Equal(t, udmi.PointsEvent{
		dbo.FieldZoneOccupancyStatus: {PresentValue: dbo.OccupancyUnoccupied},
	}, sent("pir"), "an undescribed sensor claims no empty room")
}
