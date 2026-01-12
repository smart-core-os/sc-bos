package opcua

import (
	"context"
	"testing"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

func TestUdmi_GetExportMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{
			"kind":"smartcore.bos.UDMI",
			"topicPrefix":"test/device",
			"points":{
				"point1":{"nodeId":"ns=2;s=Tag1","name":"Temperature"}
			}
		}`),
	}

	udmi, err := newUdmi("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newUdmi() error = %v", err)
	}

	msg, err := udmi.GetExportMessage(t.Context(), &udmipb.GetExportMessageRequest{})
	if err != nil {
		t.Errorf("GetExportMessage() error = %v", err)
	}
	if msg == nil {
		t.Fatal("GetExportMessage() returned nil")
	}
	if msg.Topic != "test/device/event/pointset" {
		t.Errorf("Topic = %v, want test/device/event/pointset", msg.Topic)
	}
}

func TestUdmi_sendUdmiMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name      string
		config    string
		nodeId    string
		value     any
		wantSent  bool
		pointName string
	}{
		{
			name: "valid message",
			config: `{
				"kind":"smartcore.bos.UDMI",
				"topicPrefix":"test/device",
				"points":{
					"temp":{"nodeId":"ns=2;s=Tag1","name":"Temperature"}
				}
			}`,
			nodeId:    "ns=2;s=Tag1",
			value:     float32(23.5),
			wantSent:  true,
			pointName: "Temperature",
		},
		{
			name: "enum conversion",
			config: `{
				"kind":"smartcore.bos.UDMI",
				"topicPrefix":"test/device",
				"points":{
					"status":{"nodeId":"ns=2;s=Status","name":"DoorStatus","enum":{"0":"CLOSED","1":"OPEN"}}
				}
			}`,
			nodeId:    "ns=2;s=Status",
			value:     int32(1),
			wantSent:  true,
			pointName: "DoorStatus",
		},
		{
			name: "unknown node id - not sent",
			config: `{
				"kind":"smartcore.bos.UDMI",
				"topicPrefix":"test/device",
				"points":{
					"temp":{"nodeId":"ns=2;s=Tag1","name":"Temperature"}
				}
			}`,
			nodeId:   "ns=2;s=UnknownTag",
			value:    float32(23.5),
			wantSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			udmi, err := newUdmi("test/device", cfg, logger)
			if err != nil {
				t.Fatalf("newUdmi() error = %v", err)
			}

			ctx := t.Context()
			nodeId, _ := ua.ParseNodeID(tt.nodeId)

			// Send message
			udmi.sendUdmiMessage(ctx, nodeId, tt.value)

			// Check if message was recorded in pointEvents
			msg, err := udmi.GetExportMessage(ctx, &udmipb.GetExportMessageRequest{})
			if err != nil {
				t.Fatalf("GetExportMessage() error = %v", err)
			}

			if tt.wantSent {
				// Message should contain the point data
				if msg.Payload == "" || msg.Payload == "{}" {
					t.Errorf("Expected payload to contain point data, got %v", msg.Payload)
				}
			}
		})
	}
}

func TestUdmi_PullControlTopics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{
			"kind":"smartcore.bos.UDMI",
			"topicPrefix":"test/device",
			"points":{
				"point1":{"nodeId":"ns=2;s=Tag1","name":"Temperature"}
			}
		}`),
	}

	udmi, err := newUdmi("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newUdmi() error = %v", err)
	}

	// Create a mock server that we can cancel
	ctx, cancel := context.WithCancel(t.Context())

	// Start pulling in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- udmi.PullControlTopics(&udmipb.PullControlTopicsRequest{}, &mockUdmiControlTopicsServer{ctx: ctx})
	}()

	// Cancel context
	cancel()

	// Should return without error
	err = <-errCh
	if err != nil && err != context.Canceled {
		t.Errorf("PullControlTopics() error = %v", err)
	}
}

func TestUdmi_OnMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{
			"kind":"smartcore.bos.UDMI",
			"topicPrefix":"test/device",
			"points":{
				"point1":{"nodeId":"ns=2;s=Tag1","name":"Temperature"}
			}
		}`),
	}

	udmi, err := newUdmi("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newUdmi() error = %v", err)
	}

	// OnMessage currently returns empty response
	resp, err := udmi.OnMessage(t.Context(), &udmipb.OnMessageRequest{})
	if err != nil {
		t.Errorf("OnMessage() error = %v", err)
	}
	if resp == nil {
		t.Error("OnMessage() returned nil")
	}
}

func TestUdmi_newUdmi_InvalidConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name:      "invalid json",
			config:    `{invalid json}`,
			wantError: true,
		},
		{
			name: "valid config",
			config: `{
				"kind":"smartcore.bos.UDMI",
				"topicPrefix":"test/device",
				"points":{
					"point1":{"nodeId":"ns=2;s=Tag1","name":"Temperature"}
				}
			}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			_, err := newUdmi("test/device", cfg, logger)
			if (err != nil) != tt.wantError {
				t.Errorf("newUdmi() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Mock server for testing PullControlTopics
type mockUdmiControlTopicsServer struct {
	udmipb.UdmiService_PullControlTopicsServer
	ctx context.Context
}

func (m *mockUdmiControlTopicsServer) Context() context.Context {
	return m.ctx
}

func (m *mockUdmiControlTopicsServer) Send(*udmipb.PullControlTopicsResponse) error {
	return nil
}
