package opcua

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
)

func TestTransport_GetTransport(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`),
	}

	transport, err := newTransport("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newTransport() error = %v", err)
	}

	state, err := transport.GetTransport(t.Context(), &gen.GetTransportRequest{})
	if err != nil {
		t.Errorf("GetTransport() error = %v", err)
	}
	if state == nil {
		t.Fatal("GetTransport() returned nil")
	}
}

func TestTransport_DescribeTransport(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Transport","loadUnit":"kg","maxLoad":1000,"speedUnit":"m/s","actualPosition":{"nodeId":"ns=2;s=Position"}}`),
	}

	transport, err := newTransport("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newTransport() error = %v", err)
	}

	support, err := transport.DescribeTransport(t.Context(), &gen.DescribeTransportRequest{})
	if err != nil {
		t.Errorf("DescribeTransport() error = %v", err)
	}
	if support.LoadUnit != "kg" {
		t.Errorf("LoadUnit = %v, want kg", support.LoadUnit)
	}
	if support.MaxLoad != 1000 {
		t.Errorf("MaxLoad = %v, want 1000", support.MaxLoad)
	}
	if support.SpeedUnit != "m/s" {
		t.Errorf("SpeedUnit = %v, want m/s", support.SpeedUnit)
	}
}

func TestTransport_handleTransportEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name       string
		config     string
		nodeId     string
		value      any
		checkField string
		wantValue  any
	}{
		{
			name:       "actual position",
			config:     `{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`,
			nodeId:     "ns=2;s=Position",
			value:      "5",
			checkField: "actualPosition",
			wantValue:  "5",
		},
		{
			name:       "actual position from int",
			config:     `{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`,
			nodeId:     "ns=2;s=Position",
			value:      int32(3),
			checkField: "actualPosition",
			wantValue:  "3",
		},
		{
			name:       "load",
			config:     `{"kind":"smartcore.bos.Transport","load":{"nodeId":"ns=2;s=Load"}}`,
			nodeId:     "ns=2;s=Load",
			value:      float32(750.5),
			checkField: "load",
			wantValue:  float32(750.5),
		},
		{
			name:       "speed",
			config:     `{"kind":"smartcore.bos.Transport","speed":{"nodeId":"ns=2;s=Speed"}}`,
			nodeId:     "ns=2;s=Speed",
			value:      float32(2.5),
			checkField: "speed",
			wantValue:  float32(2.5),
		},
		{
			name:       "moving direction with enum",
			config:     `{"kind":"smartcore.bos.Transport","movingDirection":{"nodeId":"ns=2;s=Direction","enum":{"1":"UP","2":"DOWN","0":"STATIONARY"}}}`,
			nodeId:     "ns=2;s=Direction",
			value:      int32(1),
			checkField: "movingDirection",
			wantValue:  gen.Transport_UP,
		},
		{
			name:       "operating mode with enum",
			config:     `{"kind":"smartcore.bos.Transport","operatingMode":{"nodeId":"ns=2;s=Mode","enum":{"0":"NORMAL","1":"EMERGENCY"}}}`,
			nodeId:     "ns=2;s=Mode",
			value:      int32(0),
			checkField: "operatingMode",
			wantValue:  gen.Transport_NORMAL,
		},
		{
			name:       "next destinations - single floor",
			config:     `{"kind":"smartcore.bos.Transport","nextDestinations":[{"type":"SingleFloor","source":{"nodeId":"ns=2;s=Next"}}]}`,
			nodeId:     "ns=2;s=Next",
			value:      int32(7),
			checkField: "nextDestinations",
			wantValue:  "7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			transport, err := newTransport("test/device", cfg, logger)
			if err != nil {
				t.Fatalf("newTransport() error = %v", err)
			}

			nodeId, _ := ua.ParseNodeID(tt.nodeId)
			transport.handleTransportEvent(nodeId, tt.value)

			state, _ := transport.GetTransport(t.Context(), &gen.GetTransportRequest{})

			switch tt.checkField {
			case "actualPosition":
				if state.ActualPosition == nil {
					t.Fatal("ActualPosition is nil")
				}
				if state.ActualPosition.Floor != tt.wantValue.(string) {
					t.Errorf("ActualPosition.Floor = %v, want %v", state.ActualPosition.Floor, tt.wantValue)
				}
			case "load":
				if state.Load == nil {
					t.Fatal("Load is nil")
				}
				if *state.Load != tt.wantValue.(float32) {
					t.Errorf("Load = %v, want %v", *state.Load, tt.wantValue)
				}
			case "speed":
				if state.Speed == nil {
					t.Fatal("Speed is nil")
				}
				if *state.Speed != tt.wantValue.(float32) {
					t.Errorf("Speed = %v, want %v", *state.Speed, tt.wantValue)
				}
			case "movingDirection":
				if state.MovingDirection != tt.wantValue.(gen.Transport_Direction) {
					t.Errorf("MovingDirection = %v, want %v", state.MovingDirection, tt.wantValue)
				}
			case "operatingMode":
				if state.OperatingMode != tt.wantValue.(gen.Transport_OperatingMode) {
					t.Errorf("OperatingMode = %v, want %v", state.OperatingMode, tt.wantValue)
				}
			case "nextDestinations":
				if len(state.NextDestinations) == 0 {
					t.Fatal("NextDestinations is empty")
				}
				if state.NextDestinations[0].Floor != tt.wantValue.(string) {
					t.Errorf("NextDestinations[0].Floor = %v, want %v", state.NextDestinations[0].Floor, tt.wantValue)
				}
			}
		})
	}
}

func TestTransport_handleTransportEvent_Doors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{
			"kind":"smartcore.bos.Transport",
			"doors":[
				{"title":"Front","status":{"nodeId":"ns=2;s=DoorStatus","enum":{"1":"OPEN","2":"CLOSED"}}}
			]
		}`),
	}

	transport, err := newTransport("test/device", cfg, logger)
	if err != nil {
		t.Fatalf("newTransport() error = %v", err)
	}

	// Check initial door setup
	state, _ := transport.GetTransport(t.Context(), &gen.GetTransportRequest{})
	if len(state.Doors) != 1 {
		t.Fatalf("Initial doors count = %v, want 1", len(state.Doors))
	}
	if state.Doors[0].Title != "Front" {
		t.Errorf("Door title = %v, want Front", state.Doors[0].Title)
	}

	// Update door status
	nodeId, _ := ua.ParseNodeID("ns=2;s=DoorStatus")
	transport.handleTransportEvent(nodeId, int32(1))

	state, _ = transport.GetTransport(t.Context(), &gen.GetTransportRequest{})
	if state.Doors[0].Status != gen.Transport_Door_OPEN {
		t.Errorf("Door status = %v, want OPEN", state.Doors[0].Status)
	}
}

func TestTransport_newTransport_InvalidConfig(t *testing.T) {
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
			name:      "valid config",
			config:    `{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			_, err := newTransport("test/device", cfg, logger)
			if (err != nil) != tt.wantError {
				t.Errorf("newTransport() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
