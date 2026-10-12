package connecttelemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/auto"
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
	require.NoError(t, a.getAllTraitImplementors(context.Background(), meterpb.TraitName, allDevices))

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
	require.NoError(t, a.getAllTraitImplementors(context.Background(), trait.OccupancySensor, allDevices))
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
	for name, use := range map[string]*resourceusepb.ResourceUse{"node1": nodeUse, "cohort": cohortUse} {
		model := resourceusepb.NewModel(resource.WithInitialValue(use))
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
	require.NoError(t, a.getAllTraitImplementors(context.Background(), resourceusepb.TraitName, allDevices))

	require.Len(t, allDevices, 1, "a gateway must not republish its cohort's resources")
	dev := allDevices["node1"]
	require.NotNil(t, dev)
	require.Len(t, dev.collectors, 1)

	// Discovery before any telemetry fetch is served from the seeded reading.
	disc := dev.buildDiscovery(time.Now())
	require.NotNil(t, disc.Pointset)
	assert.Equal(t, udmi.MetadataPoint{Units: "percent"}, disc.Pointset.Points["cpu_utilization"])
	assert.Equal(t, udmi.MetadataPoint{Units: "bytes", Ref: "/"}, disc.Pointset.Points["disk_root_usage"])

	// Telemetry is stamped with now, as ResourceUse carries no timestamp.
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	ev, ok := dev.buildTelemetry(context.Background(), now, logger)
	require.True(t, ok)
	assert.Equal(t, now, ev.Timestamp)
	assert.Equal(t, udmi.PointsEvent{
		"cpu_utilization":       {PresentValue: float32(12.5)},
		"disk_root_utilization": {PresentValue: float32(25)},
		"disk_root_usage":       {PresentValue: uint64(100)},
		"disk_root_limit":       {PresentValue: uint64(400)},
	}, ev.Points)
}
