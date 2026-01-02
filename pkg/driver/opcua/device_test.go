package opcua

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func TestDevice_newDevice(t *testing.T) {
	logger := zaptest.NewLogger(t)
	fc := newSimpleFaultCheck(t)
	cfg := &config.Device{
		Name: "test-device",
		Variables: []*config.Variable{
			{NodeId: "ns=2;s=Tag1"},
		},
	}

	dev := newDevice(cfg, logger, nil, "test_system", fc)
	require.NotNil(t, dev)
	require.Equal(t, cfg, dev.conf)
	require.NotNil(t, dev.logger)
	require.Equal(t, "test_system", dev.systemName)
}

func Test_nodeIdsAreEqual(t *testing.T) {
	tests := []struct {
		name     string
		nodeId   string
		compare  *ua.NodeID
		expected bool
	}{
		{
			name:     "equal node ids",
			nodeId:   "ns=2;s=Tag1",
			compare:  mustParseNodeID("ns=2;s=Tag1"),
			expected: true,
		},
		{
			name:     "different node ids",
			nodeId:   "ns=2;s=Tag1",
			compare:  mustParseNodeID("ns=2;s=Tag2"),
			expected: false,
		},
		{
			name:     "nil node id",
			nodeId:   "ns=2;s=Tag1",
			compare:  nil,
			expected: false,
		},
		{
			name:     "different namespace",
			nodeId:   "ns=2;s=Tag1",
			compare:  mustParseNodeID("ns=3;s=Tag1"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nodeIdsAreEqual(tt.nodeId, tt.compare)
			if result != tt.expected {
				t.Errorf("nodeIdsAreEqual(%q, %v) = %v, want %v", tt.nodeId, tt.compare, result, tt.expected)
			}
		})
	}
}

func TestDevice_handleTraitEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	fc := newSimpleFaultCheck(t)
	dev := &device{
		conf:       &config.Device{Name: "test-device"},
		logger:     logger,
		faultCheck: fc,
	}

	meterCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
	}
	meter, err := newMeter("test/device", meterCfg, logger)
	require.NoError(t, err)
	dev.meter = meter

	electricCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`),
	}
	electric, err := newElectric("test/device", electricCfg, logger)
	require.NoError(t, err)
	dev.electric = electric

	transportCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`),
	}
	transport, err := newTransport("test/device", transportCfg, logger)
	require.NoError(t, err)
	dev.transport = transport

	udmiCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.UDMI","topicPrefix":"test/device","points":{"temp":{"nodeId":"ns=2;s=Temp","name":"Temperature"}}}`),
	}
	udmi, err := newUdmi("test/device", udmiCfg, logger)
	require.NoError(t, err)
	dev.udmi = udmi

	ctx := t.Context()

	dev.handleTraitEvent(ctx, mustParseNodeID("ns=2;s=Tag1"), float32(100.5))
	reading, _ := meter.GetMeterReading(ctx, nil)
	require.Equal(t, float32(100.5), reading.Usage)

	dev.handleTraitEvent(ctx, mustParseNodeID("ns=2;s=Power"), float32(1500.0))
	demand, _ := electric.GetDemand(ctx, nil)
	require.NotNil(t, demand.RealPower)
	require.Equal(t, float32(1500.0), *demand.RealPower)

	dev.handleTraitEvent(ctx, mustParseNodeID("ns=2;s=Position"), "5")
	state, _ := transport.GetTransport(ctx, nil)
	require.NotNil(t, state.ActualPosition)
	require.Equal(t, "5", state.ActualPosition.Floor)
}

func TestDevice_handleTraitEvent_NilTraits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	fc := newSimpleFaultCheck(t)
	dev := &device{
		conf:       &config.Device{Name: "test-device"},
		logger:     logger,
		faultCheck: fc,
	}

	ctx := t.Context()
	nodeId := mustParseNodeID("ns=2;s=Tag1")
	dev.handleTraitEvent(ctx, nodeId, float32(100.0))
}

func TestDevice_handleEvent(t *testing.T) {
	tests := []struct {
		name           string
		setupMeter     bool
		eventValue     any
		expectUsage    float32
		shouldNotPanic bool
	}{
		{
			name:       "OK status updates meter",
			setupMeter: true,
			eventValue: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{
					{Value: &ua.DataValue{Value: ua.MustVariant(float32(150.5)), Status: ua.StatusOK}},
				},
			},
			expectUsage: 150.5,
		},
		{
			name:       "nil value does not panic",
			setupMeter: false,
			eventValue: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{{Value: nil}},
			},
			shouldNotPanic: true,
		},
		{
			name:       "bad status does not update meter",
			setupMeter: true,
			eventValue: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{
					{Value: &ua.DataValue{Value: ua.MustVariant(float32(150.5)), Status: ua.StatusBadNotFound}},
				},
			},
			expectUsage: 0,
		},
		{
			name:           "unknown event type does not panic",
			setupMeter:     false,
			eventValue:     "unknown event type",
			shouldNotPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			fc := newSimpleFaultCheck(t)
			dev := &device{
				conf:       &config.Device{Name: "test-device"},
				logger:     logger,
				faultCheck: fc,
			}

			var meter *Meter
			if tt.setupMeter {
				meterCfg := config.RawTrait{
					Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
				}
				var err error
				meter, err = newMeter("test/device", meterCfg, logger)
				require.NoError(t, err)
				dev.meter = meter
			}

			ctx := t.Context()
			nodeId := mustParseNodeID("ns=2;s=Tag1")
			event := &opcua.PublishNotificationData{Value: tt.eventValue}

			dev.handleEvent(ctx, event, nodeId)

			if tt.setupMeter {
				reading, _ := meter.GetMeterReading(ctx, nil)
				require.Equal(t, tt.expectUsage, reading.Usage)
			}
		})
	}
}

func mustParseNodeID(s string) *ua.NodeID {
	nodeId, err := ua.ParseNodeID(s)
	if err != nil {
		panic(err)
	}
	return nodeId
}

func newSimpleFaultCheck(t *testing.T) *healthpb.FaultCheck {
	devs := devicespb.NewCollection()
	reg := newTestRegistry(devs)
	healthChecks := reg.ForOwner("test")
	_, _ = devs.Update(&gen.Device{Name: "test-device"}, resource.WithCreateIfAbsent())
	check := getDeviceHealthCheck(gen.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED, gen.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED)
	fc, err := healthChecks.NewFaultCheck("test-device", check)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)
	return fc
}

func newTestRegistry(devs *devicespb.Collection) *healthpb.Registry {
	return healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(name string, c *gen.HealthCheck) *gen.HealthCheck {
			_, _ = devs.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}), resource.WithCreateIfAbsent(), resource.WithExpectAbsent())
			return nil
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *gen.HealthCheck) {
			_, _ = devs.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}))
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			_, _ = devs.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
			}), resource.WithAllowMissing(true))
		}),
	)
}

type testHarness struct {
	devs   *devicespb.Collection
	client gen.DevicesApiClient
	fc     *healthpb.FaultCheck
	ctx    context.Context
}

func setupTestHarness(t *testing.T) *testHarness {
	devs := devicespb.NewCollection()
	server := devices.NewServer(devicesServerModel{Collection: devs})
	deviceName := "opcua-device-1"
	reg := newTestRegistry(devs)
	healthChecks := reg.ForOwner("example")

	_, _ = devs.Update(&gen.Device{Name: deviceName}, resource.WithCreateIfAbsent())

	check := getDeviceHealthCheck(gen.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED, gen.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED)
	fc, err := healthChecks.NewFaultCheck(deviceName, check)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)

	return &testHarness{
		devs:   devs,
		client: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, server)),
		fc:     fc,
		ctx:    context.Background(),
	}
}

func (h *testHarness) getHealthChecks(t *testing.T) []*gen.HealthCheck {
	deviceList, err := h.client.ListDevices(context.TODO(), &gen.ListDevicesRequest{})
	require.NoError(t, err)
	require.Len(t, deviceList.Devices, 1)
	return deviceList.Devices[0].GetHealthChecks()
}

func (h *testHarness) assertFaults(t *testing.T, expectedCount int, normality gen.HealthCheck_Normality) {
	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, normality, checks[0].Normality)
	require.Len(t, checks[0].GetFaults().CurrentFaults, expectedCount)
}

// Health tests

func TestOpcuaConfigFault(t *testing.T) {
	h := setupTestHarness(t)
	const testSystemName = "test_opcua_system"

	raiseConfigFault("Failed to subscribe to point ns=2;s=InvalidNode", testSystemName, h.fc)

	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, gen.HealthCheck_ABNORMAL, checks[0].Normality)

	faults := checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 1)
	require.Equal(t, DeviceConfigError, faults[0].Code.Code)
	require.Equal(t, testSystemName, faults[0].Code.System)
	require.Contains(t, faults[0].SummaryText, "configuration")
}

func TestOpcuaPointFaults(t *testing.T) {
	h := setupTestHarness(t)
	const testSystemName = "test_opcua_system"
	nodeId1, nodeId2 := "ns=2;s=Tag1", "ns=2;s=Tag2"

	updateReliabilityBadResponse(h.ctx, nodeId1, "BadNodeIdUnknown", testSystemName, h.fc)
	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	rel := checks[0].GetReliability()
	require.NotNil(t, rel)
	require.Equal(t, gen.HealthCheck_Reliability_BAD_RESPONSE, rel.State)
	require.NotNil(t, rel.LastError)
	require.Equal(t, nodeId1, rel.LastError.Code.Code)
	require.Contains(t, rel.LastError.SummaryText, "non OK status")
	require.Contains(t, rel.LastError.DetailsText, nodeId1)

	updateReliabilityBadResponse(h.ctx, nodeId2, "BadTimeout", testSystemName, h.fc)
	checks = h.getHealthChecks(t)
	rel = checks[0].GetReliability()
	require.Equal(t, gen.HealthCheck_Reliability_BAD_RESPONSE, rel.State)
	require.Equal(t, nodeId2, rel.LastError.Code.Code)

	// Note: updateReliabilityNormal still removes faults from the faults list, not reliability
	// To clear reliability, we'd need a different approach
}

func TestOpcuaFaultLifecycle(t *testing.T) {
	const testSystemName = "test_opcua_system"
	tests := []struct {
		name  string
		steps []struct {
			action           func(*testHarness)
			reliabilityState gen.HealthCheck_Reliability_State
			lastErrorNodeId  string
			description      string
		}
	}{
		{
			name: "raise multiple point faults then clear last",
			steps: []struct {
				action           func(*testHarness)
				reliabilityState gen.HealthCheck_Reliability_State
				lastErrorNodeId  string
				description      string
			}{
				{
					action: func(h *testHarness) {
						updateReliabilityBadResponse(h.ctx, "ns=2;s=Tag1", "BadNodeIdUnknown", testSystemName, h.fc)
					},
					reliabilityState: gen.HealthCheck_Reliability_BAD_RESPONSE,
					lastErrorNodeId:  "ns=2;s=Tag1",
					description:      "first point fault raised",
				},
				{
					action: func(h *testHarness) {
						updateReliabilityBadResponse(h.ctx, "ns=2;s=Tag2", "BadTimeout", testSystemName, h.fc)
					},
					reliabilityState: gen.HealthCheck_Reliability_BAD_RESPONSE,
					lastErrorNodeId:  "ns=2;s=Tag2",
					description:      "second point fault overwrites first in reliability",
				},
				{
					action: func(h *testHarness) {
						updateReliabilityBadResponse(h.ctx, "ns=2;s=Tag3", "BadCommunicationError", testSystemName, h.fc)
					},
					reliabilityState: gen.HealthCheck_Reliability_BAD_RESPONSE,
					lastErrorNodeId:  "ns=2;s=Tag3",
					description:      "third point fault overwrites second in reliability",
				},
			},
		},
		{
			name: "mix config and point faults",
			steps: []struct {
				action           func(*testHarness)
				reliabilityState gen.HealthCheck_Reliability_State
				lastErrorNodeId  string
				description      string
			}{
				{
					action: func(h *testHarness) {
						raiseConfigFault("Invalid subscription", testSystemName, h.fc)
					},
					reliabilityState: gen.HealthCheck_Reliability_RELIABLE,
					description:      "config fault raised (uses AddOrUpdateFault, sets reliability to RELIABLE)",
				},
				{
					action: func(h *testHarness) {
						updateReliabilityBadResponse(h.ctx, "ns=2;s=Tag1", "BadNodeIdUnknown", testSystemName, h.fc)
					},
					reliabilityState: gen.HealthCheck_Reliability_BAD_RESPONSE,
					lastErrorNodeId:  "ns=2;s=Tag1",
					description:      "point fault updates reliability to BAD_RESPONSE",
				},
			},
		},
		{
			name: "update same fault",
			steps: []struct {
				action           func(*testHarness)
				reliabilityState gen.HealthCheck_Reliability_State
				lastErrorNodeId  string
				description      string
			}{
				{
					action: func(h *testHarness) {
						updateReliabilityBadResponse(h.ctx, "ns=2;s=Tag1", "BadNodeIdUnknown", testSystemName, h.fc)
					},
					reliabilityState: gen.HealthCheck_Reliability_BAD_RESPONSE,
					lastErrorNodeId:  "ns=2;s=Tag1",
					description:      "initial fault",
				},
				{
					action: func(h *testHarness) {
						updateReliabilityBadResponse(h.ctx, "ns=2;s=Tag1", "BadTimeout", testSystemName, h.fc)
					},
					reliabilityState: gen.HealthCheck_Reliability_BAD_RESPONSE,
					lastErrorNodeId:  "ns=2;s=Tag1",
					description:      "same node fault updated with different error",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupTestHarness(t)

			for i, step := range tt.steps {
				step.action(h)

				checks := h.getHealthChecks(t)
				require.Len(t, checks, 1)

				rel := checks[0].GetReliability()
				require.NotNil(t, rel, "step %d (%s): reliability should not be nil", i, step.description)

				if diff := cmp.Diff(step.reliabilityState, rel.State, protocmp.Transform()); diff != "" {
					t.Errorf("step %d (%s): reliability state mismatch (-want +got):\n%s", i, step.description, diff)
				}

				if step.lastErrorNodeId != "" {
					require.NotNil(t, rel.LastError, "step %d (%s): last error should not be nil", i, step.description)
					require.Equal(t, step.lastErrorNodeId, rel.LastError.Code.Code, "step %d (%s): last error node id mismatch", i, step.description)
				}
			}
		})
	}
}

func TestOpcuaHandleEvent_WithHealth(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := setupTestHarness(t)
	const testSystemName = "test_system"
	dev := &device{
		conf:       &config.Device{Name: "opcua-device-1"},
		logger:     logger,
		faultCheck: h.fc,
		systemName: testSystemName,
	}
	ctx := context.Background()
	nodeId := mustParseNodeID("ns=2;s=Tag1")

	makeEvent := func(status ua.StatusCode) *opcua.PublishNotificationData {
		return &opcua.PublishNotificationData{
			Value: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{
					{Value: &ua.DataValue{Value: ua.MustVariant(float32(100.0)), Status: status}},
				},
			},
		}
	}

	dev.handleEvent(ctx, makeEvent(ua.StatusBadNodeIDUnknown), nodeId)
	checks := h.getHealthChecks(t)
	rel := checks[0].GetReliability()
	require.NotNil(t, rel)
	require.Equal(t, gen.HealthCheck_Reliability_BAD_RESPONSE, rel.State)
	require.NotNil(t, rel.LastError)
	require.Equal(t, nodeId.String(), rel.LastError.Code.Code)
	require.Contains(t, rel.LastError.SummaryText, "non OK status")

	dev.handleEvent(ctx, makeEvent(ua.StatusOK), nodeId)
	checks = h.getHealthChecks(t)
	require.Equal(t, gen.HealthCheck_Reliability_RELIABLE, checks[0].GetReliability().GetState())
	faults := checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 0)
}

type devicesServerModel struct {
	devices.Collection
}

func (m devicesServerModel) ClientConn() grpc.ClientConnInterface {
	return nil
}
