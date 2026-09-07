package opcua

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
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

	dev := newDevice(cfg, logger, nil, fc, nil)
	require.NotNil(t, dev)
	require.Equal(t, cfg, dev.conf)
	require.NotNil(t, dev.logger)
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
	dev.eventHandlers = append(dev.eventHandlers, meter)

	electricCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`),
	}
	electric, err := newElectric("test/device", electricCfg, logger)
	require.NoError(t, err)
	dev.eventHandlers = append(dev.eventHandlers, electric)

	transportCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`),
	}
	transport, err := newTransport("test/device", transportCfg, logger)
	require.NoError(t, err)
	dev.eventHandlers = append(dev.eventHandlers, transport)

	udmiCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.UDMI","topicPrefix":"test/device","points":{"temp":{"nodeId":"ns=2;s=Temp","name":"Temperature"}}}`),
	}
	udmi, err := newUdmi("test/device", udmiCfg, logger)
	require.NoError(t, err)
	dev.eventHandlers = append(dev.eventHandlers, udmi)

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
			// 0x480 is Good with the Overflow info bit set, which the driver used to reject.
			name:       "good status with info bits updates meter",
			setupMeter: true,
			eventValue: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{
					{Value: &ua.DataValue{Value: ua.MustVariant(float32(150.5)), Status: ua.StatusCode(0x480)}},
				},
			},
			expectUsage: 150.5,
		},
		{
			name:       "named good status updates meter",
			setupMeter: true,
			eventValue: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{
					{Value: &ua.DataValue{Value: ua.MustVariant(float32(150.5)), Status: ua.StatusGoodCallAgain}},
				},
			},
			expectUsage: 150.5,
		},
		{
			name:       "uncertain status still updates meter",
			setupMeter: true,
			eventValue: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{
					{Value: &ua.DataValue{Value: ua.MustVariant(float32(150.5)), Status: ua.StatusUncertainSimulatedValue}},
				},
			},
			expectUsage: 150.5,
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
				dev.eventHandlers = append(dev.eventHandlers, meter)
			}

			ctx := t.Context()
			nodeId := mustParseNodeID("ns=2;s=Tag1")
			event := &opcua.PublishNotificationData{Value: tt.eventValue}

			dev.handleEvent(ctx, event, fixedLookup(nodeId))

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
	_, _ = devs.Update(&devicespb.Device{Name: "test-device"}, resource.WithCreateIfAbsent())
	check := getDeviceHealthCheck(healthpb.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED, healthpb.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED)
	fc, err := healthChecks.NewFaultCheck("test-device", check)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)
	return fc
}

func newTestRegistry(devs *devicespb.Collection) *healthpb.Registry {
	return healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(name string, c *healthpb.HealthCheck) *healthpb.HealthCheck {
			_, _ = devs.Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*devicespb.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}), resource.WithCreateIfAbsent(), resource.WithExpectAbsent())
			return nil
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *healthpb.HealthCheck) {
			_, _ = devs.Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*devicespb.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}))
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			_, _ = devs.Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*devicespb.Device)
				dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
			}), resource.WithAllowMissing(true))
		}),
	)
}

type testHarness struct {
	devs   *devicespb.Collection
	client devicespb.DevicesApiClient
	fc     *healthpb.FaultCheck
	ctx    context.Context
}

func setupTestHarness(t *testing.T) *testHarness {
	devs := devicespb.NewCollection()
	server := devices.NewServer(devicesServerModel{Collection: devs})
	deviceName := "opcua-device-1"
	reg := newTestRegistry(devs)
	healthChecks := reg.ForOwner("example")

	_, _ = devs.Update(&devicespb.Device{Name: deviceName}, resource.WithCreateIfAbsent())

	check := getDeviceHealthCheck(healthpb.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED, healthpb.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED)
	fc, err := healthChecks.NewFaultCheck(deviceName, check)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)

	return &testHarness{
		devs:   devs,
		client: devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, server)),
		fc:     fc,
		ctx:    context.Background(),
	}
}

func (h *testHarness) getHealthChecks(t *testing.T) []*healthpb.HealthCheck {
	deviceList, err := h.client.ListDevices(context.TODO(), &devicespb.ListDevicesRequest{})
	require.NoError(t, err)
	require.Len(t, deviceList.Devices, 1)
	return deviceList.Devices[0].GetHealthChecks()
}

func TestOpcuaConfigFault(t *testing.T) {
	h := setupTestHarness(t)

	raiseConfigFault("Failed to subscribe to point ns=2;s=InvalidNode", h.fc)

	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	require.Equal(t, healthpb.HealthCheck_ABNORMAL, checks[0].Normality)

	faults := checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 1)
	require.Equal(t, DeviceConfigError, faults[0].Code.Code)
	require.Equal(t, SystemName, faults[0].Code.System)
	require.Contains(t, faults[0].SummaryText, "configuration")
}

func TestOpcuaPointFaults(t *testing.T) {
	h := setupTestHarness(t)
	nodeId1, nodeId2 := "ns=2;s=Tag1", "ns=2;s=Tag2"

	setPointReadNotOk(h.ctx, nodeId1, ua.StatusBadNodeIDUnknown, h.fc)
	checks := h.getHealthChecks(t)
	require.Len(t, checks, 1)
	rel := checks[0].GetReliability()
	require.NotNil(t, rel)
	require.Equal(t, healthpb.HealthCheck_Reliability_BAD_RESPONSE, rel.State)
	require.NotNil(t, rel.LastError)
	require.Contains(t, rel.LastError.SummaryText, "non OK status")
	require.Contains(t, rel.LastError.DetailsText, nodeId1)

	setPointReadNotOk(h.ctx, nodeId2, ua.StatusBadTimeout, h.fc)
	checks = h.getHealthChecks(t)
	rel = checks[0].GetReliability()
	require.Equal(t, healthpb.HealthCheck_Reliability_BAD_RESPONSE, rel.State)
	require.Contains(t, rel.LastError.DetailsText, nodeId2)
}

func TestOpcuaFaultLifecycle(t *testing.T) {
	tests := []struct {
		name  string
		steps []struct {
			action             func(*testHarness)
			reliabilityState   healthpb.HealthCheck_Reliability_State
			expectNodeInDetail string
			description        string
		}
	}{
		{
			name: "raise multiple point faults then clear last",
			steps: []struct {
				action             func(*testHarness)
				reliabilityState   healthpb.HealthCheck_Reliability_State
				expectNodeInDetail string
				description        string
			}{
				{
					action: func(h *testHarness) {
						setPointReadNotOk(h.ctx, "ns=2;s=Tag1", ua.StatusBadNodeIDUnknown, h.fc)
					},
					reliabilityState:   healthpb.HealthCheck_Reliability_BAD_RESPONSE,
					expectNodeInDetail: "ns=2;s=Tag1",
					description:        "first point fault raised",
				},
				{
					action: func(h *testHarness) {
						setPointReadNotOk(h.ctx, "ns=2;s=Tag2", ua.StatusBadTimeout, h.fc)
					},
					reliabilityState:   healthpb.HealthCheck_Reliability_BAD_RESPONSE,
					expectNodeInDetail: "ns=2;s=Tag2",
					description:        "second point fault overwrites first in reliability",
				},
				{
					action: func(h *testHarness) {
						setPointReadNotOk(h.ctx, "ns=2;s=Tag3", ua.StatusBadCommunicationError, h.fc)
					},
					reliabilityState:   healthpb.HealthCheck_Reliability_BAD_RESPONSE,
					expectNodeInDetail: "ns=2;s=Tag3",
					description:        "third point fault overwrites second in reliability",
				},
			},
		},
		{
			name: "mix config and point faults",
			steps: []struct {
				action             func(*testHarness)
				reliabilityState   healthpb.HealthCheck_Reliability_State
				expectNodeInDetail string
				description        string
			}{
				{
					action: func(h *testHarness) {
						raiseConfigFault("Invalid subscription", h.fc)
					},
					reliabilityState: healthpb.HealthCheck_Reliability_RELIABLE,
					description:      "config fault raised (uses AddOrUpdateFault, sets reliability to RELIABLE)",
				},
				{
					action: func(h *testHarness) {
						setPointReadNotOk(h.ctx, "ns=2;s=Tag1", ua.StatusBadNodeIDUnknown, h.fc)
					},
					reliabilityState:   healthpb.HealthCheck_Reliability_BAD_RESPONSE,
					expectNodeInDetail: "ns=2;s=Tag1",
					description:        "point fault updates reliability to BAD_RESPONSE",
				},
			},
		},
		{
			name: "update same fault",
			steps: []struct {
				action             func(*testHarness)
				reliabilityState   healthpb.HealthCheck_Reliability_State
				expectNodeInDetail string
				description        string
			}{
				{
					action: func(h *testHarness) {
						setPointReadNotOk(h.ctx, "ns=2;s=Tag1", ua.StatusBadNodeIDUnknown, h.fc)
					},
					reliabilityState:   healthpb.HealthCheck_Reliability_BAD_RESPONSE,
					expectNodeInDetail: "ns=2;s=Tag1",
					description:        "initial fault",
				},
				{
					action: func(h *testHarness) {
						setPointReadNotOk(h.ctx, "ns=2;s=Tag1", ua.StatusBadTimeout, h.fc)
					},
					reliabilityState:   healthpb.HealthCheck_Reliability_BAD_RESPONSE,
					expectNodeInDetail: "ns=2;s=Tag1",
					description:        "same node fault updated with different error",
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

				if step.expectNodeInDetail != "" {
					require.NotNil(t, rel.LastError, "step %d (%s): last error should not be nil", i, step.description)
					require.Equal(t, SystemName, rel.LastError.Code.System, "step %d (%s): error system should be SystemName constant", i, step.description)
					require.Contains(t, rel.LastError.DetailsText, step.expectNodeInDetail, "step %d (%s): details should contain node ID", i, step.description)
				}
			}
		})
	}
}

// makeStatusEvent is one value arriving under the client handle of the item that produced it,
// which is all a notification says about which node it belongs to.
func makeStatusEvent(handle uint32, status ua.StatusCode) *opcua.PublishNotificationData {
	return &opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{
			MonitoredItems: []*ua.MonitoredItemNotification{
				{ClientHandle: handle, Value: &ua.DataValue{Value: ua.MustVariant(float32(100.0)), Status: status}},
			},
		},
	}
}

// fixedLookup resolves every client handle to the same node, for the dispatch tests where
// which node a value came from is not what is under test.
func fixedLookup(nodeId *ua.NodeID) nodeLookup {
	return func(uint32) (*ua.NodeID, bool) { return nodeId, true }
}

func TestOpcuaHandleEvent_WithHealth(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := setupTestHarness(t)
	dev := &device{
		conf:       &config.Device{Name: "opcua-device-1"},
		logger:     logger,
		faultCheck: h.fc,
	}
	ctx := context.Background()
	nodeId := mustParseNodeID("ns=2;s=Tag1")

	dev.handleEvent(ctx, makeStatusEvent(1, ua.StatusBadNodeIDUnknown), fixedLookup(nodeId))
	checks := h.getHealthChecks(t)
	rel := checks[0].GetReliability()
	require.NotNil(t, rel)
	require.Equal(t, healthpb.HealthCheck_Reliability_BAD_RESPONSE, rel.State)
	require.NotNil(t, rel.LastError)

	require.Equal(t, SystemName, rel.LastError.Code.System)
	require.Contains(t, rel.LastError.SummaryText, "non OK status")
	require.Contains(t, rel.LastError.DetailsText, nodeId.String())

	dev.handleEvent(ctx, makeStatusEvent(1, ua.StatusOK), fixedLookup(nodeId))
	checks = h.getHealthChecks(t)
	require.Equal(t, healthpb.HealthCheck_Reliability_RELIABLE, checks[0].GetReliability().GetState())

	faults := checks[0].GetFaults().GetCurrentFaults()
	require.Len(t, faults, 0)
}

// TestOpcuaHandleEvent_Severity checks the driver reports health from the severity bits of the
// status code alone. A Good code carrying info bits, 0x480 being the Overflow bit a busy
// subscription queue sets, must not look like a read failure.
func TestOpcuaHandleEvent_Severity(t *testing.T) {
	tests := []struct {
		name    string
		status  ua.StatusCode
		want    healthpb.HealthCheck_Reliability_State
		wantErr bool // an error is attached to the reliability report
	}{
		{name: "ok", status: ua.StatusOK, want: healthpb.HealthCheck_Reliability_RELIABLE},
		{name: "good with overflow info bit", status: ua.StatusCode(0x480), want: healthpb.HealthCheck_Reliability_RELIABLE},
		{name: "named good", status: ua.StatusGoodCallAgain, want: healthpb.HealthCheck_Reliability_RELIABLE},
		{name: "uncertain", status: ua.StatusUncertainSimulatedValue, want: healthpb.HealthCheck_Reliability_UNRELIABLE, wantErr: true},
		{name: "bad", status: ua.StatusBadNodeIDUnknown, want: healthpb.HealthCheck_Reliability_BAD_RESPONSE, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			h := setupTestHarness(t)
			nodeId := mustParseNodeID("ns=2;s=Tag1")

			meterCfg := config.RawTrait{
				Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
			}
			meter, err := newMeter("opcua-device-1", meterCfg, logger)
			require.NoError(t, err)

			dev := &device{
				conf:          &config.Device{Name: "opcua-device-1"},
				logger:        logger,
				faultCheck:    h.fc,
				eventHandlers: []EventHandler{meter},
			}

			dev.handleEvent(h.ctx, makeStatusEvent(1, tt.status), fixedLookup(nodeId))

			rel := h.getHealthChecks(t)[0].GetReliability()
			require.NotNil(t, rel)
			require.Equal(t, tt.want, rel.State)

			// only a Bad status withholds the value from the traits
			wantUsage := float32(100)
			if statusIsBad(tt.status) {
				wantUsage = 0
			}
			reading, err := meter.GetMeterReading(h.ctx, nil)
			require.NoError(t, err)
			require.Equal(t, wantUsage, reading.Usage)

			if !tt.wantErr {
				return
			}
			require.NotNil(t, rel.LastError)
			require.Equal(t, SystemName, rel.LastError.Code.System)
			require.Equal(t, fmt.Sprintf("0x%X", uint32(tt.status)), rel.LastError.Code.Code)
			require.Contains(t, rel.LastError.DetailsText, nodeId.String())
		})
	}
}

type devicesServerModel struct {
	devices.Collection
}

func (m devicesServerModel) ClientConn() grpc.ClientConnInterface {
	return nil
}

// fakeSubscriber hands out a scripted sequence of subscriptions and records the calls,
// standing in for *Client so the retry behaviour can be driven without an OPC UA server.
type fakeSubscriber struct {
	mu sync.Mutex
	// errs is the outcome of each successive NewSubscription call, a nil entry meaning a
	// subscription is created. The last entry repeats once the script runs out.
	errs []error
	// outcomes is the scripted status for a node id on each successive occasion it is asked
	// about, a nil entry meaning the item was created. The last entry repeats, so
	// {StatusBadTimeout, nil} fails once and then works forever. Kept here rather than on the
	// subscription so a script carries across a resubscribe.
	outcomes map[string][]error
	seen     map[string]int // how many times each node has been asked about, over all subs

	subs  []*fakePointSub
	calls int
}

func newFakeSubscriber(outcomes map[string][]error, errs ...error) *fakeSubscriber {
	return &fakeSubscriber{errs: errs, outcomes: outcomes, seen: make(map[string]int)}
}

func (f *fakeSubscriber) NewSubscription(_ context.Context) (pointSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	i := f.calls
	f.calls++
	if len(f.errs) > 0 {
		if i >= len(f.errs) {
			i = len(f.errs) - 1
		}
		if err := f.errs[i]; err != nil {
			return nil, err
		}
	}
	sub := &fakePointSub{
		owner:  f,
		notify: make(chan *opcua.PublishNotificationData),
		nodes:  make(map[uint32]*ua.NodeID),
	}
	f.subs = append(f.subs, sub)
	return sub, nil
}

func (f *fakeSubscriber) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// subAt returns the subscription handed out by successful call number i, counting from zero.
func (f *fakeSubscriber) subAt(i int) *fakePointSub {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.subs[i]
}

func (f *fakeSubscriber) subCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.subs)
}

// outcome consumes the next scripted result for nodeId, wrapped the way Client wraps it so
// that subscribeErrIsPermanent classifies it from the status code inside.
func (f *fakeSubscriber) outcome(nodeId *ua.NodeID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := nodeId.String()
	script := f.outcomes[key]
	asked := f.seen[key]
	f.seen[key]++
	if len(script) == 0 {
		return nil
	}
	err := script[min(asked, len(script)-1)]
	if err == nil {
		return nil
	}
	return monitorErr(nodeId, err)
}

// fakePointSub is one scripted subscription: it answers Monitor from its subscriber's script,
// hands out client handles as items are created, and records everything it was asked.
type fakePointSub struct {
	owner *fakeSubscriber

	mu         sync.Mutex
	calls      [][]string // node ids of each successive Monitor call, in call order
	cancels    int
	nextHandle uint32
	nodes      map[uint32]*ua.NodeID

	notify chan *opcua.PublishNotificationData
}

func (f *fakePointSub) Monitor(_ context.Context, nodeIds ...*ua.NodeID) []monitorResult {
	results := make([]monitorResult, 0, len(nodeIds))
	asked := make([]string, 0, len(nodeIds))
	for _, nodeId := range nodeIds {
		asked = append(asked, nodeId.String())
		err := f.owner.outcome(nodeId)
		if err == nil {
			f.mu.Lock()
			f.nextHandle++
			f.nodes[f.nextHandle] = nodeId
			f.mu.Unlock()
		}
		results = append(results, monitorResult{NodeId: nodeId, Err: err})
	}
	f.mu.Lock()
	f.calls = append(f.calls, asked)
	f.mu.Unlock()
	return results
}

func (f *fakePointSub) Notifications() <-chan *opcua.PublishNotificationData {
	return f.notify
}

func (f *fakePointSub) Node(handle uint32) (*ua.NodeID, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	nodeId, ok := f.nodes[handle]
	return nodeId, ok
}

func (f *fakePointSub) Cancel(context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancels++
}

// send delivers a notification to whoever is pumping this subscription, blocking until it is
// taken so that a test knows it has been dispatched.
func (f *fakePointSub) send(event *opcua.PublishNotificationData) {
	f.notify <- event
}

// monitorCalls returns the node ids of each successive Monitor call.
func (f *fakePointSub) monitorCalls() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *fakePointSub) cancelCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cancels
}

// handleFor returns the client handle the subscription gave nodeId, or zero when it has no
// item for it.
func (f *fakePointSub) handleFor(nodeId string) uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	for handle, n := range f.nodes {
		if n.String() == nodeId {
			return handle
		}
	}
	return 0
}

// faultRecorder keeps the latest state of a health check as the registry commits it, so a test
// can read the current faults without going through the devices API. The harness above starts
// gRPC goroutines of its own, which do not belong inside a synctest bubble.
type faultRecorder struct {
	mu    sync.Mutex
	check *healthpb.HealthCheck
}

func (r *faultRecorder) set(c *healthpb.HealthCheck) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.check = c
}

// fault returns the current fault carrying code, or nil when there is none.
func (r *faultRecorder) fault(code string) *healthpb.HealthCheck_Error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.check.GetFaults().GetCurrentFaults() {
		if f.GetCode().GetCode() == code {
			return f
		}
	}
	return nil
}

func newRecordedFaultCheck(t *testing.T) (*healthpb.FaultCheck, *faultRecorder) {
	t.Helper()
	rec := &faultRecorder{}
	reg := healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(_ string, c *healthpb.HealthCheck) *healthpb.HealthCheck {
			rec.set(c)
			return nil
		}),
		healthpb.WithOnCheckUpdate(func(_ string, c *healthpb.HealthCheck) { rec.set(c) }),
	)
	check := getDeviceHealthCheck(healthpb.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED, healthpb.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED)
	fc, err := reg.ForOwner("test").NewFaultCheck("test-device", check)
	require.NoError(t, err)
	t.Cleanup(fc.Dispose)
	return fc, rec
}

// newTestDevice builds a device wired to sub, with a meter reading the first node, and returns
// the recorder watching its fault check. The stagger is zeroed so the retry delays are the
// only thing on the clock.
func newTestDevice(t *testing.T, sub subscriber, nodeIds ...string) (*device, *Meter, *faultRecorder) {
	t.Helper()
	logger := zaptest.NewLogger(t)
	fc, rec := newRecordedFaultCheck(t)

	cfg := &config.Device{Name: "test-device"}
	for _, nodeId := range nodeIds {
		cfg.Variables = append(cfg.Variables, &config.Variable{NodeId: nodeId, ParsedNodeId: mustParseNodeID(nodeId)})
	}

	meter, err := newMeter("test-device", config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"` + nodeIds[0] + `"}}`),
	}, logger)
	require.NoError(t, err)

	dev := newDevice(cfg, logger, sub, fc, nil)
	dev.eventHandlers = append(dev.eventHandlers, meter)
	dev.maxStagger = 0
	return dev, meter, rec
}

func goSubscribe(ctx context.Context, dev *device) <-chan error {
	done := make(chan error, 1)
	go func() { done <- dev.subscribe(ctx) }()
	return done
}

// requireCalls checks how many subscriptions the device has asked for. One per device is the
// point of the exercise, so most of these tests are really assertions about this number.
func requireCalls(t *testing.T, sub *fakeSubscriber, want int) {
	t.Helper()
	if got := sub.callCount(); got != want {
		t.Fatalf("NewSubscription called %d times, want %d", got, want)
	}
}

// requireMonitored checks the node ids of each successive Monitor call on one subscription.
func requireMonitored(t *testing.T, sub *fakePointSub, want [][]string) {
	t.Helper()
	require.Equal(t, want, sub.monitorCalls(), "the nodes monitored, per Monitor call")
}

func requireSubscribeReturns(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("subscribe() = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("subscribe() did not return after its ctx was cancelled")
	}
}

// recordingHandler captures every dispatch, for the tests about which node a value belongs to.
type recordingHandler struct {
	mu     sync.Mutex
	events []dispatch
}

type dispatch struct {
	node  string
	value any
}

func (r *recordingHandler) handleEvent(_ context.Context, node *ua.NodeID, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, dispatch{node: node.String(), value: value})
}

func (r *recordingHandler) dispatched() []dispatch {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]dispatch, len(r.events))
	copy(out, r.events)
	return out
}

// mapLookup resolves client handles from a fixed map, standing in for pointSub.Node.
func mapLookup(nodes map[uint32]string) nodeLookup {
	return func(handle uint32) (*ua.NodeID, bool) {
		nodeId, ok := nodes[handle]
		if !ok {
			return nil, false
		}
		return mustParseNodeID(nodeId), true
	}
}

// TestDevice_handleEvent_demux is the whole reason a client handle is stamped on each item: one
// subscription carries every point on the device, and the handle is the only thing on a
// notification that says which node a value came from.
func TestDevice_handleEvent_demux(t *testing.T) {
	const nodeA, nodeB = "ns=2;s=A", "ns=2;s=B"
	rec := &recordingHandler{}
	dev := &device{
		conf:          &config.Device{Name: "test-device"},
		logger:        zaptest.NewLogger(t),
		faultCheck:    newSimpleFaultCheck(t),
		eventHandlers: []EventHandler{rec},
	}

	dev.handleEvent(t.Context(), &opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{
			MonitoredItems: []*ua.MonitoredItemNotification{
				{ClientHandle: 2, Value: &ua.DataValue{Value: ua.MustVariant(float32(2)), Status: ua.StatusOK}},
				{ClientHandle: 1, Value: &ua.DataValue{Value: ua.MustVariant(float32(1)), Status: ua.StatusOK}},
			},
		},
	}, mapLookup(map[uint32]string{1: nodeA, 2: nodeB}))

	want := []dispatch{{node: nodeB, value: float32(2)}, {node: nodeA, value: float32(1)}}
	require.Equal(t, want, rec.dispatched(), "each value should reach the node its handle names")
}

// TestDevice_handleEvent_unknownHandle checks a notification we cannot attribute is dropped.
// There is nothing on the wire saying which node it came from, so there is no point to fault,
// and the device's own points are unaffected by the server reporting an item we do not know:
// calling it a read failure would blame every point on the device for it.
func TestDevice_handleEvent_unknownHandle(t *testing.T) {
	rec := &recordingHandler{}
	fc, faults := newRecordedFaultCheck(t)
	dev := &device{
		conf:          &config.Device{Name: "test-device"},
		logger:        zaptest.NewLogger(t),
		faultCheck:    fc,
		eventHandlers: []EventHandler{rec},
	}

	// a value and a nil value, since the handle is resolved before the nil check
	dev.handleEvent(t.Context(), &opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{
			MonitoredItems: []*ua.MonitoredItemNotification{
				{ClientHandle: 99, Value: &ua.DataValue{Value: ua.MustVariant(float32(1)), Status: ua.StatusOK}},
				{ClientHandle: 99, Value: nil},
			},
		},
	}, mapLookup(nil))

	require.Empty(t, rec.dispatched(), "a value we cannot attribute must not be dispatched")
	require.Nil(t, faults.fault(PointSubscribeError))
	require.Nil(t, faults.fault(DeviceConfigError))
}

// TestDevice_logUnknownHandle_thins checks the logging ladder. A server that keeps reporting an
// item we do not know about would otherwise repeat the same line every publishing cycle for as
// long as the config lives, drowning out whatever else is going on.
func TestDevice_logUnknownHandle_thins(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	dev := &device{conf: &config.Device{Name: "test-device"}, logger: zap.New(core)}

	for range 5 {
		dev.logUnknownHandle(7)
	}
	require.Equal(t, 3, logs.FilterLevelExact(zap.WarnLevel).Len(),
		"two lines then one saying it is reducing, however many follow")
	require.Zero(t, logs.FilterLevelExact(zap.DebugLevel).Len())

	for range 95 {
		dev.logUnknownHandle(7)
	}
	require.Equal(t, 3, logs.FilterLevelExact(zap.WarnLevel).Len())
	require.Equal(t, 1, logs.FilterLevelExact(zap.DebugLevel).Len(), "one line per hundred after that")
}

// TestDevice_subscribe_partialFailureKeepsSubscription is the case that decides the whole shape
// of the retry: one point of many failing must not disturb the ones that are working. With a
// subscription per device, tearing it down to retry that point would take every sibling with it.
func TestDevice_subscribe_partialFailureKeepsSubscription(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const good, bad = "ns=2;s=Tag1", "ns=2;s=Tag2"
		sub := newFakeSubscriber(map[string][]error{bad: {ua.StatusBadTimeout}})
		dev, meter, rec := newTestDevice(t, sub, good, bad)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		requireCalls(t, sub, 1)
		requireMonitored(t, sub.subAt(0), [][]string{{good, bad}})
		require.Zero(t, sub.subAt(0).cancelCount(), "a subscription with points delivering on it is not an orphan")

		fault := rec.fault(PointSubscribeError)
		require.NotNil(t, fault, "the point being retried should raise a %s fault", PointSubscribeError)
		require.Contains(t, fault.DetailsText, bad)
		require.NotContains(t, fault.DetailsText, good, "a point that is delivering is not a fault")
		require.Nil(t, rec.fault(DeviceConfigError), "being retried is not being misconfigured")

		// and the good point is delivering, which is the thing that used to be lost
		sub.subAt(0).send(makeStatusEvent(sub.subAt(0).handleFor(good), ua.StatusOK))
		synctest.Wait()
		reading, err := meter.GetMeterReading(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, float32(100), reading.Usage)

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_retriesStragglerOnLiveSubscription is the other half of that: the point
// that failed has to rejoin its siblings, and it does so by being monitored onto the
// subscription they are already delivering on rather than by starting a new one.
func TestDevice_subscribe_retriesStragglerOnLiveSubscription(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const good, late = "ns=2;s=Tag1", "ns=2;s=Tag2"
		sub := newFakeSubscriber(map[string][]error{late: {ua.StatusBadTimeout, nil}})
		dev, meter, rec := newTestDevice(t, sub, good, late)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		require.NotNil(t, rec.fault(PointSubscribeError))

		// the straggler waits before its first retry: it was asked about a moment ago
		time.Sleep(subscribeRetryInitial)
		synctest.Wait()

		requireCalls(t, sub, 1)
		requireMonitored(t, sub.subAt(0), [][]string{{good, late}, {late}})
		require.Zero(t, sub.subAt(0).cancelCount())
		require.Nil(t, rec.fault(PointSubscribeError), "the fault should clear once the point is monitored")
		require.NotZero(t, sub.subAt(0).handleFor(late), "the straggler should have an item on the live subscription")

		// the point that was working all along has not been disturbed
		sub.subAt(0).send(makeStatusEvent(sub.subAt(0).handleFor(good), ua.StatusOK))
		synctest.Wait()
		reading, err := meter.GetMeterReading(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, float32(100), reading.Usage)

		// and the retry loop is done, rather than idling for the life of the device
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		requireMonitored(t, sub.subAt(0), [][]string{{good, late}, {late}})

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_stragglerBacksOff pins the straggler's own ramp. It cannot share the
// device's, which describes a subscription that will not open at all, but it has to ramp for
// the same reason: a point the server keeps refusing should not be asked for every two seconds
// all night.
func TestDevice_subscribe_stragglerBacksOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const good, late = "ns=2;s=Tag1", "ns=2;s=Tag2"
		sub := newFakeSubscriber(map[string][]error{
			late: {ua.StatusBadTimeout, ua.StatusBadTimeout, nil},
		})
		dev, _, _ := newTestDevice(t, sub, good, late)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		requireMonitored(t, sub.subAt(0), [][]string{{good, late}, {late}})

		// ramped by half, so 3s rather than another 2s
		time.Sleep(3*time.Second - time.Millisecond)
		synctest.Wait()
		requireMonitored(t, sub.subAt(0), [][]string{{good, late}, {late}})

		time.Sleep(time.Millisecond)
		synctest.Wait()
		requireMonitored(t, sub.subAt(0), [][]string{{good, late}, {late}, {late}})
		requireCalls(t, sub, 1)

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_stragglerGivenUp covers a straggler whose next answer says it can never
// work. It moves to the config fault and stops being asked about, which also ends the retry
// goroutine, so a device whose points have settled leaves nothing of its own running.
func TestDevice_subscribe_stragglerGivenUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const good, doomed = "ns=2;s=Tag1", "ns=2;s=Tag2"
		sub := newFakeSubscriber(map[string][]error{
			doomed: {ua.StatusBadTimeout, ua.StatusBadNodeIDUnknown},
		})
		dev, _, rec := newTestDevice(t, sub, good, doomed)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		require.NotNil(t, rec.fault(PointSubscribeError))

		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		fault := rec.fault(DeviceConfigError)
		require.NotNil(t, fault, "a point the server says can never work is a config fault")
		require.Contains(t, fault.DetailsText, doomed)
		require.Nil(t, rec.fault(PointSubscribeError), "a point we gave up on is not one being retried")

		// never asked about again, and the subscription its sibling is using stays up
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		requireMonitored(t, sub.subAt(0), [][]string{{good, doomed}, {doomed}})
		requireCalls(t, sub, 1)
		require.Zero(t, sub.subAt(0).cancelCount())

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_cancelsOrphanSubscription checks a subscription nothing is delivering on
// is torn down before the device backs off.
//
// Left behind it would sit in the client folding into the client-wide publish timeout, and
// gopcua spawns a permanently blocked notify goroutine for every stale subscription on every
// transport error - one leak per resubscribe.
func TestDevice_subscribe_cancelsOrphanSubscription(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeA, nodeB = "ns=2;s=Tag1", "ns=2;s=Tag2"
		timeouts := []error{ua.StatusBadTimeout, ua.StatusBadTimeout, nil}
		sub := newFakeSubscriber(map[string][]error{nodeA: timeouts, nodeB: timeouts})
		dev, _, rec := newTestDevice(t, sub, nodeA, nodeB)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		requireCalls(t, sub, 1)
		require.Equal(t, 1, sub.subAt(0).cancelCount(), "a subscription with nothing on it is an orphan")
		fault := rec.fault(PointSubscribeError)
		require.NotNil(t, fault)
		require.Contains(t, fault.DetailsText, nodeA)
		require.Contains(t, fault.DetailsText, nodeB)

		// a whole new subscription per attempt, each orphan cancelled once
		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		requireCalls(t, sub, 2)
		require.Equal(t, 1, sub.subAt(1).cancelCount())

		time.Sleep(3 * time.Second)
		synctest.Wait()
		requireCalls(t, sub, 3)
		require.Zero(t, sub.subAt(2).cancelCount(), "the attempt that worked must keep its subscription")
		require.Nil(t, rec.fault(PointSubscribeError))

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_givesUpWhenEveryPointRefused checks a device the server has refused
// outright stops asking, without subscribe returning.
//
// Returning is the part that matters: applyConfig reads grp.Wait returning as the config being
// finished and takes it as the cue to close the client every other device is still using.
func TestDevice_subscribe_givesUpWhenEveryPointRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeA, nodeB = "ns=2;s=MissingA", "ns=2;s=MissingB"
		unknown := []error{ua.StatusBadNodeIDUnknown}
		sub := newFakeSubscriber(map[string][]error{nodeA: unknown, nodeB: unknown})
		dev, _, rec := newTestDevice(t, sub, nodeA, nodeB)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		// both points named rather than the second overwriting the first: faults are keyed on
		// system and code, so one fault naming its points is the only shape that survives
		fault := rec.fault(DeviceConfigError)
		require.NotNil(t, fault)
		require.Contains(t, fault.DetailsText, nodeA)
		require.Contains(t, fault.DetailsText, nodeB)
		require.Nil(t, rec.fault(PointSubscribeError))
		require.Equal(t, 1, sub.subAt(0).cancelCount())

		// one attempt and no more, however long we wait
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		requireCalls(t, sub, 1)
		select {
		case err := <-done:
			t.Fatalf("subscribe() returned %v while its ctx was still live", err)
		default:
		}

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_noConfiguredPoints checks a device with nothing to monitor asks the
// server for nothing at all, rather than opening a subscription it would put no items on.
func TestDevice_subscribe_noConfiguredPoints(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sub := newFakeSubscriber(nil)
		fc, rec := newRecordedFaultCheck(t)
		dev := newDevice(&config.Device{Name: "test-device"}, zaptest.NewLogger(t), sub, fc, nil)
		dev.maxStagger = 0

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		requireCalls(t, sub, 0)
		require.Nil(t, rec.fault(PointSubscribeError))
		require.Nil(t, rec.fault(DeviceConfigError))

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_retriesFailedSubscription covers the failure that is nothing to do with
// any one point: the subscription itself will not open. Every point is reported as waiting on
// it, since none of them can be told apart at that stage.
func TestDevice_subscribe_retriesFailedSubscription(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeId = "ns=2;s=Tag1"
		sub := newFakeSubscriber(nil, ua.StatusBadTooManySubscriptions, nil)
		dev, _, rec := newTestDevice(t, sub, nodeId)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		requireCalls(t, sub, 1)
		require.Zero(t, sub.subCount(), "no subscription was created, so there is none to cancel")
		fault := rec.fault(PointSubscribeError)
		require.NotNil(t, fault)
		require.Contains(t, fault.DetailsText, nodeId)
		// a capacity code clears as load drops, so it is retried rather than given up on
		require.Nil(t, rec.fault(DeviceConfigError))

		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		requireCalls(t, sub, 2)
		requireMonitored(t, sub.subAt(0), [][]string{{nodeId}})
		require.Nil(t, rec.fault(PointSubscribeError))

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_resubscribesWhenSubscriptionEnds covers a subscription that worked and
// then died. gopcua never closes the notification channel, so that arrives as a
// StatusChangeNotification carrying a Bad status, which handleEvent used to discard as an
// unhandled event, leaving every point on the device silently dead.
//
// It also pins two things the batching brought with it: the dead subscription is cancelled
// exactly once, and a point already given up on stays given up on across the resubscribe.
func TestDevice_subscribe_resubscribesWhenSubscriptionEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const good, doomed = "ns=2;s=Tag1", "ns=2;s=Tag2"
		sub := newFakeSubscriber(map[string][]error{
			doomed: {ua.StatusBadTimeout, ua.StatusBadNodeIDUnknown},
		})
		dev, _, rec := newTestDevice(t, sub, good, doomed)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		// let the straggler retry turn the second point into a permanent failure
		synctest.Wait()
		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		require.NotNil(t, rec.fault(DeviceConfigError))

		// now the server ends the subscription under us
		sub.subAt(0).send(&opcua.PublishNotificationData{
			Value: &ua.StatusChangeNotification{Status: ua.StatusBadTimeout},
		})
		synctest.Wait()
		require.Equal(t, 1, sub.subAt(0).cancelCount(), "the dead subscription is cancelled exactly once")
		requireCalls(t, sub, 1) // a backoff away, not immediate
		require.NotNil(t, rec.fault(PointSubscribeError), "the points are waiting on a new subscription")

		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		requireCalls(t, sub, 2)
		// the point we gave up on is not asked about again, and its fault stands
		requireMonitored(t, sub.subAt(1), [][]string{{good}})
		require.NotNil(t, rec.fault(DeviceConfigError))
		require.Nil(t, rec.fault(PointSubscribeError))

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_resetsBackoffAfterGrace checks a subscription that ran for a while
// resubscribes from the initial delay rather than carrying a ramp built from an older problem.
// The grace now measures the whole device, so one flapping point can no longer keep the ramp up.
func TestDevice_subscribe_resetsBackoffAfterGrace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeId = "ns=2;s=Tag1"
		// two failures first so the ramp has reached 4.5s by the time a subscription opens.
		// Without them a reset and a normal first delay are the same number.
		sub := newFakeSubscriber(nil, ua.StatusBadTimeout, ua.StatusBadTimeout, nil)
		dev, _, _ := newTestDevice(t, sub, nodeId)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		time.Sleep(3 * time.Second)
		synctest.Wait()
		requireCalls(t, sub, 3)

		// let it run long enough to count as working, then have the server end it
		time.Sleep(resubscribeGrace)
		sub.subAt(0).send(&opcua.PublishNotificationData{
			Value: &ua.StatusChangeNotification{Status: ua.StatusBadTimeout},
		})
		synctest.Wait()
		requireCalls(t, sub, 3)

		time.Sleep(subscribeRetryInitial - time.Millisecond)
		synctest.Wait()
		requireCalls(t, sub, 3)

		// subscribeRetryInitial, not the 4.5s the ramp had reached
		time.Sleep(time.Millisecond)
		synctest.Wait()
		requireCalls(t, sub, 4)

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_dedupesNodeIds checks a config listing the same node twice monitors it
// once. Two entries used to mean two subscriptions to the same node, which was merely wasteful;
// on one subscription it would mean two items delivering the same value under two handles.
func TestDevice_subscribe_dedupesNodeIds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeA, nodeB = "ns=2;s=Tag1", "ns=2;s=Tag2"
		sub := newFakeSubscriber(nil)
		dev, _, _ := newTestDevice(t, sub, nodeA, nodeB, nodeA)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		requireMonitored(t, sub.subAt(0), [][]string{{nodeA, nodeB}})

		cancel()
		requireSubscribeReturns(t, done)
	})
}
