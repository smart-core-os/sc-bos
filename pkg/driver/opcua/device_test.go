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
	"go.uber.org/zap/zaptest"
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

func makeStatusEvent(status ua.StatusCode) *opcua.PublishNotificationData {
	return &opcua.PublishNotificationData{
		Value: &ua.DataChangeNotification{
			MonitoredItems: []*ua.MonitoredItemNotification{
				{Value: &ua.DataValue{Value: ua.MustVariant(float32(100.0)), Status: status}},
			},
		},
	}
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

	dev.handleEvent(ctx, makeStatusEvent(ua.StatusBadNodeIDUnknown), nodeId)
	checks := h.getHealthChecks(t)
	rel := checks[0].GetReliability()
	require.NotNil(t, rel)
	require.Equal(t, healthpb.HealthCheck_Reliability_BAD_RESPONSE, rel.State)
	require.NotNil(t, rel.LastError)

	require.Equal(t, SystemName, rel.LastError.Code.System)
	require.Contains(t, rel.LastError.SummaryText, "non OK status")
	require.Contains(t, rel.LastError.DetailsText, nodeId.String())

	dev.handleEvent(ctx, makeStatusEvent(ua.StatusOK), nodeId)
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

			dev.handleEvent(h.ctx, makeStatusEvent(tt.status), nodeId)

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

// fakeSubscriber hands out a scripted sequence of Subscribe outcomes and records the calls,
// standing in for *Client so the retry behaviour can be driven without an OPC UA server.
type fakeSubscriber struct {
	mu sync.Mutex
	// errs is the outcome of each successive call, a nil entry meaning success. The last entry
	// repeats once the script runs out, so a script ending in nil keeps succeeding.
	errs  []error
	calls []string                              // node ids, in call order
	chans []chan *opcua.PublishNotificationData // one per successful call, in call order
}

func newFakeSubscriber(errs ...error) *fakeSubscriber {
	return &fakeSubscriber{errs: errs}
}

func (f *fakeSubscriber) Subscribe(_ context.Context, nodeId *ua.NodeID) (<-chan *opcua.PublishNotificationData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	i := len(f.calls)
	f.calls = append(f.calls, nodeId.String())
	if len(f.errs) > 0 {
		if i >= len(f.errs) {
			i = len(f.errs) - 1
		}
		if err := f.errs[i]; err != nil {
			return nil, err
		}
	}
	ch := make(chan *opcua.PublishNotificationData)
	f.chans = append(f.chans, ch)
	return ch, nil
}

func (f *fakeSubscriber) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// chanAt returns the channel handed out by successful Subscribe number i, counting from zero.
func (f *fakeSubscriber) chanAt(i int) chan *opcua.PublishNotificationData {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.chans[i]
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

func requireCalls(t *testing.T, sub *fakeSubscriber, want int) {
	t.Helper()
	if got := sub.callCount(); got != want {
		t.Fatalf("Subscribe called %d times, want %d", got, want)
	}
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

// eventually polls cond until it holds, for the tests whose goroutines run on the real clock.
func eventually(t *testing.T, cond func() bool, msg string, args ...any) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf(msg, args...)
}

// TestDevice_subscribe_retriesTransientFailure is the reported fault: a point that browses
// fine fails to subscribe with StatusBadTimeout, which is our own request timeout expiring
// against a slow server rather than anything wrong with the node. It used to be abandoned for
// the lifetime of the config.
func TestDevice_subscribe_retriesTransientFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeId = "ns=2;s=Tag1"
		sub := newFakeSubscriber(ua.StatusBadTimeout, ua.StatusBadTimeout, nil)
		dev, meter, _ := newTestDevice(t, sub, nodeId)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		requireCalls(t, sub, 1)

		// WithBackoff(2s, 5m) ramps by half each time, so the delays are 2s then 3s
		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		requireCalls(t, sub, 2)

		time.Sleep(3 * time.Second)
		synctest.Wait()
		requireCalls(t, sub, 3)

		// the third attempt worked, so values reach the traits again
		sub.chanAt(0) <- makeStatusEvent(ua.StatusOK)
		synctest.Wait()
		reading, err := meter.GetMeterReading(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, float32(100), reading.Usage, "the point should deliver values once it subscribes")

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_givesUpOnPermanentFailure covers the other half of the split: a node id
// the server does not know is never going to appear, so asking again forever would be noise.
func TestDevice_subscribe_givesUpOnPermanentFailure(t *testing.T) {
	const nodeId = "ns=2;s=MissingTag"
	sub := newFakeSubscriber(ua.StatusBadNodeIDUnknown)
	dev, _, rec := newTestDevice(t, sub, nodeId)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := goSubscribe(ctx, dev)

	eventually(t, func() bool { return rec.fault(DeviceConfigError) != nil },
		"no %s fault was raised for a point the server rejected", DeviceConfigError)

	fault := rec.fault(DeviceConfigError)
	require.Contains(t, fault.DetailsText, nodeId, "the fault should name the point")
	require.Contains(t, fault.DetailsText, ua.StatusBadNodeIDUnknown.Error())
	// one attempt and no more. The first retry would be subscribeRetryInitial away, so this
	// also says it stopped rather than merely not having got there yet.
	require.Equal(t, 1, sub.callCount())
	require.Nil(t, rec.fault(PointSubscribeError), "a point we gave up on is not one being retried")

	cancel()
	requireSubscribeReturns(t, done)
}

// TestDevice_subscribe_survivesAllPointsFailing is a regression test for the downstream
// symptom of the old behaviour. subscribe used to return once it had no live subscriptions
// left, and applyConfig reads that as the config being finished and takes it as the cue to
// close the client every other device on the connection is still using.
func TestDevice_subscribe_survivesAllPointsFailing(t *testing.T) {
	const nodeA, nodeB = "ns=2;s=MissingA", "ns=2;s=MissingB"
	sub := newFakeSubscriber(ua.StatusBadNodeIDUnknown)
	dev, _, rec := newTestDevice(t, sub, nodeA, nodeB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := goSubscribe(ctx, dev)

	eventually(t, func() bool { return sub.callCount() == 2 }, "both points should have been attempted")

	select {
	case err := <-done:
		t.Fatalf("subscribe() returned %v while its ctx was still live", err)
	case <-time.After(100 * time.Millisecond):
	}

	// both points are named rather than the second overwriting the first: faults are keyed on
	// system and code, so one fault naming its points is the only shape that survives
	fault := rec.fault(DeviceConfigError)
	require.NotNil(t, fault)
	require.Contains(t, fault.DetailsText, nodeA)
	require.Contains(t, fault.DetailsText, nodeB)

	cancel()
	requireSubscribeReturns(t, done)
}

// TestDevice_subscribe_clearsFaultOnSuccess checks a point being retried says so, and stops
// saying so once it works. Nothing in this driver used to remove a fault it had raised.
func TestDevice_subscribe_clearsFaultOnSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeId = "ns=2;s=Tag1"
		sub := newFakeSubscriber(ua.StatusBadTimeout, nil)
		dev, _, rec := newTestDevice(t, sub, nodeId)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		fault := rec.fault(PointSubscribeError)
		require.NotNil(t, fault, "a point being retried should raise a %s fault", PointSubscribeError)
		require.Contains(t, fault.DetailsText, nodeId)
		require.Contains(t, fault.DetailsText, ua.StatusBadTimeout.Error())
		// being retried is not being misconfigured, and saying it is sends whoever reads the
		// fault hunting a mistake in the config that is not there
		require.Nil(t, rec.fault(DeviceConfigError))

		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		require.Nil(t, rec.fault(PointSubscribeError), "the fault should be removed once the point subscribes")

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_resubscribesWhenSubscriptionEnds covers a subscription that worked and
// then died. gopcua never closes the notification channel, so that arrives as a
// StatusChangeNotification carrying a Bad status, which handleEvent used to discard as an
// unhandled event, leaving the point silently dead.
//
// It also pins the ResetBackoff: a subscription that ran for a while demonstrably works, so
// the retry starts from the initial delay rather than carrying a ramp built from an older
// problem.
func TestDevice_subscribe_resubscribesWhenSubscriptionEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const nodeId = "ns=2;s=Tag1"
		// two failures first so the backoff has ramped to 4.5s by the time a subscription is
		// established. Without them a reset and a normal first delay are the same number.
		sub := newFakeSubscriber(ua.StatusBadTimeout, ua.StatusBadTimeout, nil)
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
		sub.chanAt(0) <- &opcua.PublishNotificationData{
			Value: &ua.StatusChangeNotification{Status: ua.StatusBadTimeout},
		}
		synctest.Wait()
		requireCalls(t, sub, 3) // a backoff away, not immediate

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
