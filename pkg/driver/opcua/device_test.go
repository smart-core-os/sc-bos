package opcua

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
)

func TestDevice_newDevice(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Device{
		Name: "test-device",
		Variables: []*config.Variable{
			{NodeId: "ns=2;s=Tag1"},
		},
	}

	dev := newDevice(cfg, logger, nil)
	if dev == nil {
		t.Fatal("newDevice() returned nil")
	}
	if dev.conf != cfg {
		t.Error("Device config not set correctly")
	}
	if dev.logger != logger {
		t.Error("Device logger not set correctly")
	}
}

func TestDevice_nodeIdsAreEqual(t *testing.T) {
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

	// Create a device with all trait types
	dev := &device{
		conf:   &config.Device{Name: "test-device"},
		logger: logger,
	}

	// Add a Meter trait
	meterCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
	}
	meter, err := newMeter("test/device", meterCfg, logger)
	if err != nil {
		t.Fatalf("newMeter() error = %v", err)
	}
	dev.meter = meter

	// Add an Electric trait
	electricCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`),
	}
	electric, err := newElectric("test/device", electricCfg, logger)
	if err != nil {
		t.Fatalf("newElectric() error = %v", err)
	}
	dev.electric = electric

	// Add a Transport trait
	transportCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`),
	}
	transport, err := newTransport("test/device", transportCfg, logger)
	if err != nil {
		t.Fatalf("newTransport() error = %v", err)
	}
	dev.transport = transport

	// Add UDMI trait
	udmiCfg := config.RawTrait{
		Raw: []byte(`{
			"kind":"smartcore.bos.UDMI",
			"topicPrefix":"test/device",
			"points":{
				"temp":{"nodeId":"ns=2;s=Temp","name":"Temperature"}
			}
		}`),
	}
	udmi, err := newUdmi("test/device", udmiCfg, logger)
	if err != nil {
		t.Fatalf("newUdmi() error = %v", err)
	}
	dev.udmi = udmi

	ctx := context.Background()

	// Test Meter event
	meterNodeId := mustParseNodeID("ns=2;s=Tag1")
	dev.handleTraitEvent(ctx, meterNodeId, float32(100.5))

	// Verify Meter was updated
	reading, _ := meter.GetMeterReading(ctx, nil)
	if reading.Usage != 100.5 {
		t.Errorf("Meter usage = %v, want 100.5", reading.Usage)
	}

	// Test Electric event
	powerNodeId := mustParseNodeID("ns=2;s=Power")
	dev.handleTraitEvent(ctx, powerNodeId, float32(1500.0))

	// Verify Electric was updated
	demand, _ := electric.GetDemand(ctx, nil)
	if demand.RealPower == nil || *demand.RealPower != 1500.0 {
		t.Errorf("Electric real power = %v, want 1500.0", demand.RealPower)
	}

	// Test Transport event
	positionNodeId := mustParseNodeID("ns=2;s=Position")
	dev.handleTraitEvent(ctx, positionNodeId, "5")

	// Verify Transport was updated
	state, _ := transport.GetTransport(ctx, nil)
	if state.ActualPosition == nil || state.ActualPosition.Floor != "5" {
		t.Errorf("Transport position = %v, want 5", state.ActualPosition)
	}
}

func TestDevice_handleTraitEvent_NilTraits(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create device with no traits
	dev := &device{
		conf:   &config.Device{Name: "test-device"},
		logger: logger,
	}

	ctx := context.Background()
	nodeId := mustParseNodeID("ns=2;s=Tag1")

	// Should not panic with nil traits
	dev.handleTraitEvent(ctx, nodeId, float32(100.0))
}

func TestDevice_handleEvent_DataChangeNotification(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create device with Meter trait
	dev := &device{
		conf:   &config.Device{Name: "test-device"},
		logger: logger,
	}

	meterCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
	}
	meter, err := newMeter("test/device", meterCfg, logger)
	if err != nil {
		t.Fatalf("newMeter() error = %v", err)
	}
	dev.meter = meter

	ctx := context.Background()
	nodeId := mustParseNodeID("ns=2;s=Tag1")

	// Create a mock DataChangeNotification
	variant, _ := ua.NewVariant(float32(150.5))
	dataValue := &ua.DataValue{
		Value:  variant,
		Status: ua.StatusOK,
	}

	notification := &ua.DataChangeNotification{
		MonitoredItems: []*ua.MonitoredItemNotification{
			{
				ClientHandle: 1,
				Value:        dataValue,
			},
		},
	}

	event := &opcua.PublishNotificationData{
		Value: notification,
	}

	dev.handleEvent(ctx, event, nodeId)

	// Verify Meter was updated
	reading, _ := meter.GetMeterReading(ctx, nil)
	if reading.Usage != 150.5 {
		t.Errorf("Meter usage = %v, want 150.5", reading.Usage)
	}
}

func TestDevice_handleEvent_NilValue(t *testing.T) {
	logger := zaptest.NewLogger(t)

	dev := &device{
		conf:   &config.Device{Name: "test-device"},
		logger: logger,
	}

	ctx := context.Background()
	nodeId := mustParseNodeID("ns=2;s=Tag1")

	// Test with nil Value in MonitoredItemNotification
	notification := &ua.DataChangeNotification{
		MonitoredItems: []*ua.MonitoredItemNotification{
			{
				ClientHandle: 1,
				Value:        nil,
			},
		},
	}

	event := &opcua.PublishNotificationData{
		Value: notification,
	}

	// Should not panic
	dev.handleEvent(ctx, event, nodeId)
}

func TestDevice_handleEvent_BadStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)

	dev := &device{
		conf:   &config.Device{Name: "test-device"},
		logger: logger,
	}

	meterCfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
	}
	meter, _ := newMeter("test/device", meterCfg, logger)
	dev.meter = meter

	ctx := context.Background()
	nodeId := mustParseNodeID("ns=2;s=Tag1")

	// Create notification with bad status
	variant, _ := ua.NewVariant(float32(150.5))
	dataValue := &ua.DataValue{
		Value:  variant,
		Status: ua.StatusBadNotFound,
	}

	notification := &ua.DataChangeNotification{
		MonitoredItems: []*ua.MonitoredItemNotification{
			{
				ClientHandle: 1,
				Value:        dataValue,
			},
		},
	}

	event := &opcua.PublishNotificationData{
		Value: notification,
	}

	dev.handleEvent(ctx, event, nodeId)

	// Verify Meter was NOT updated
	reading, _ := meter.GetMeterReading(ctx, nil)
	if reading.Usage != 0 {
		t.Errorf("Meter usage = %v, want 0 (should not update on bad status)", reading.Usage)
	}
}

func TestDevice_handleEvent_UnknownEventType(t *testing.T) {
	logger := zaptest.NewLogger(t)

	dev := &device{
		conf:   &config.Device{Name: "test-device"},
		logger: logger,
	}

	ctx := context.Background()
	nodeId := mustParseNodeID("ns=2;s=Tag1")

	// Unknown event type
	event := &opcua.PublishNotificationData{
		Value: "unknown event type",
	}

	// Should log warning but not panic
	dev.handleEvent(ctx, event, nodeId)
}

// Helper function to parse node ID without error handling
func mustParseNodeID(s string) *ua.NodeID {
	nodeId, err := ua.ParseNodeID(s)
	if err != nil {
		panic(err)
	}
	return nodeId
}
