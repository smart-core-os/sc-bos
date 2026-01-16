package config

import (
	"strings"
	"testing"
)

func TestValueSource_Validate(t *testing.T) {
	tests := []struct {
		name      string
		vs        *ValueSource
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid value source",
			vs:        &ValueSource{NodeId: "ns=2;s=Tag1"},
			fieldName: "test",
			wantErr:   false,
		},
		{
			name:      "nil value source",
			vs:        nil,
			fieldName: "test",
			wantErr:   false,
		},
		{
			name:      "empty nodeId",
			vs:        &ValueSource{NodeId: ""},
			fieldName: "testField",
			wantErr:   true,
			errMsg:    "testField: nodeId is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.vs.Validate(tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestMeterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     MeterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid meter config",
			cfg: MeterConfig{
				Unit:  "kWh",
				Usage: &ValueSource{NodeId: "ns=2;s=Tag1"},
			},
			wantErr: false,
		},
		{
			name: "missing usage",
			cfg: MeterConfig{
				Unit: "kWh",
			},
			wantErr: true,
			errMsg:  "meter trait: usage is required",
		},
		{
			name: "usage with empty nodeId",
			cfg: MeterConfig{
				Unit:  "kWh",
				Usage: &ValueSource{NodeId: ""},
			},
			wantErr: true,
			errMsg:  "meter usage: nodeId is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestElectricConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ElectricConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid electric config with real power",
			cfg: ElectricConfig{
				Demand: &ElectricDemandConfig{
					ElectricPhaseConfig: &ElectricPhaseConfig{
						RealPower: &ValueSource{NodeId: "ns=2;s=RealPower"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid electric config with multiple fields",
			cfg: ElectricConfig{
				Demand: &ElectricDemandConfig{
					ElectricPhaseConfig: &ElectricPhaseConfig{
						RealPower:     &ValueSource{NodeId: "ns=2;s=RealPower"},
						ApparentPower: &ValueSource{NodeId: "ns=2;s=ApparentPower"},
						PowerFactor:   &ValueSource{NodeId: "ns=2;s=PowerFactor"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "missing demand",
			cfg:     ElectricConfig{},
			wantErr: true,
			errMsg:  "electric trait: demand is required",
		},
		{
			name: "demand with no fields",
			cfg: ElectricConfig{
				Demand: &ElectricDemandConfig{},
			},
			wantErr: true,
			errMsg:  "electric demand: at least one power measurement field must be configured",
		},
		{
			name: "field with empty nodeId",
			cfg: ElectricConfig{
				Demand: &ElectricDemandConfig{
					ElectricPhaseConfig: &ElectricPhaseConfig{
						RealPower: &ValueSource{NodeId: ""},
					},
				},
			},
			wantErr: true,
			errMsg:  "electric demand realPower: nodeId is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestTransportConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     TransportConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid transport config with position",
			cfg: TransportConfig{
				ActualPosition: &ValueSource{NodeId: "ns=2;s=Position"},
			},
			wantErr: false,
		},
		{
			name: "valid transport config with doors",
			cfg: TransportConfig{
				Doors: []*Door{
					{Title: "Front", Status: &ValueSource{NodeId: "ns=2;s=DoorStatus"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid transport config with multiple fields",
			cfg: TransportConfig{
				ActualPosition:  &ValueSource{NodeId: "ns=2;s=Position"},
				Speed:           &ValueSource{NodeId: "ns=2;s=Speed"},
				MovingDirection: &ValueSource{NodeId: "ns=2;s=Direction"},
			},
			wantErr: false,
		},
		{
			name:    "no fields configured",
			cfg:     TransportConfig{},
			wantErr: true,
			errMsg:  "transport trait: at least one field must be configured",
		},
		{
			name: "field with empty nodeId",
			cfg: TransportConfig{
				Speed: &ValueSource{NodeId: ""},
			},
			wantErr: true,
			errMsg:  "transport speed: nodeId is required",
		},
		{
			name: "door with empty nodeId",
			cfg: TransportConfig{
				Doors: []*Door{
					{Title: "Front", Status: &ValueSource{NodeId: ""}},
				},
			},
			wantErr: true,
			errMsg:  "transport door[0] status: nodeId is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestUdmiConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     UdmiConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid udmi config",
			cfg: UdmiConfig{
				TopicPrefix: "test/topic",
				Points: map[string]*ValueSource{
					"point1": {NodeId: "ns=2;s=Tag1"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid udmi config with multiple points",
			cfg: UdmiConfig{
				TopicPrefix: "test/topic",
				Points: map[string]*ValueSource{
					"point1": {NodeId: "ns=2;s=Tag1"},
					"point2": {NodeId: "ns=2;s=Tag2"},
				},
			},
			wantErr: false,
		},
		{
			name: "no points configured",
			cfg: UdmiConfig{
				TopicPrefix: "test/topic",
			},
			wantErr: true,
			errMsg:  "udmi trait: at least one point must be configured",
		},
		{
			name: "point with empty nodeId",
			cfg: UdmiConfig{
				TopicPrefix: "test/topic",
				Points: map[string]*ValueSource{
					"point1": {NodeId: ""},
				},
			},
			wantErr: true,
			errMsg:  "udmi point 'point1': nodeId is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidateDeviceTraits(t *testing.T) {
	tests := []struct {
		name    string
		device  Device
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid device with meter trait",
			device: Device{
				Name: "test-device",
				Variables: []*Variable{
					{NodeId: "ns=2;s=Tag1"},
				},
				Traits: []RawTrait{
					{
						Trait: Trait{Kind: "smartcore.bos.Meter"},
						Raw:   []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "nodeId not in device variables",
			device: Device{
				Name: "test-device",
				Variables: []*Variable{
					{NodeId: "ns=2;s=Tag1"},
				},
				Traits: []RawTrait{
					{
						Trait: Trait{Kind: "smartcore.bos.Meter"},
						Raw:   []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag99"}}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "references nodeId 'ns=2;s=Tag99' which is not in device variables list",
		},
		{
			name: "invalid meter config - missing usage",
			device: Device{
				Name: "test-device",
				Variables: []*Variable{
					{NodeId: "ns=2;s=Tag1"},
				},
				Traits: []RawTrait{
					{
						Trait: Trait{Kind: "smartcore.bos.Meter"},
						Raw:   []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh"}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "meter trait: usage is required",
		},
		{
			name: "valid device with electric trait",
			device: Device{
				Name: "test-device",
				Variables: []*Variable{
					{NodeId: "ns=2;s=RealPower"},
				},
				Traits: []RawTrait{
					{
						Trait: Trait{Kind: "smartcore.traits.Electric"},
						Raw:   []byte(`{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=RealPower"}}}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid device with transport trait",
			device: Device{
				Name: "test-device",
				Variables: []*Variable{
					{NodeId: "ns=2;s=Position"},
				},
				Traits: []RawTrait{
					{
						Trait: Trait{Kind: "smartcore.bos.Transport"},
						Raw:   []byte(`{"kind":"smartcore.bos.Transport","actualPosition":{"nodeId":"ns=2;s=Position"}}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid device with udmi trait",
			device: Device{
				Name: "test-device",
				Variables: []*Variable{
					{NodeId: "ns=2;s=Tag1"},
				},
				Traits: []RawTrait{
					{
						Trait: Trait{Kind: "smartcore.bos.UDMI"},
						Raw:   []byte(`{"kind":"smartcore.bos.UDMI","topicPrefix":"test/","points":{"point1":{"nodeId":"ns=2;s=Tag1"}}}`),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDeviceTraits(&tt.device)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDeviceTraits() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateDeviceTraits() error = %v, should contain %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestHealthConfig_Validate(t *testing.T) {
	normalValue := 100.0
	tests := []struct {
		name    string
		config  HealthConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with all fields",
			config: HealthConfig{
				Checks: []HealthCheck{
					{
						Id:          "temp-check",
						DisplayName: "Temperature Check",
						Description: "Monitors temperature",
						ErrorCode:   "TEMP_ERROR",
						NormalValue: &normalValue,
						ValueSource: ValueSource{
							NodeId: "ns=2;s=Temp",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			config: HealthConfig{
				Checks: []HealthCheck{
					{
						DisplayName: "Temperature Check",
						Description: "Monitors temperature",
						ErrorCode:   "TEMP_ERROR",
						NormalValue: &normalValue,
					},
				},
			},
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name: "missing displayName",
			config: HealthConfig{
				Checks: []HealthCheck{
					{
						Id:          "temp-check",
						Description: "Monitors temperature",
						ErrorCode:   "TEMP_ERROR",
						NormalValue: &normalValue,
					},
				},
			},
			wantErr: true,
			errMsg:  "displayName is required",
		},
		{
			name: "missing description",
			config: HealthConfig{
				Checks: []HealthCheck{
					{
						Id:          "temp-check",
						DisplayName: "Temperature Check",
						ErrorCode:   "TEMP_ERROR",
						NormalValue: &normalValue,
					},
				},
			},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name: "missing errorCode",
			config: HealthConfig{
				Checks: []HealthCheck{
					{
						Id:          "temp-check",
						DisplayName: "Temperature Check",
						Description: "Monitors temperature",
						NormalValue: &normalValue,
					},
				},
			},
			wantErr: true,
			errMsg:  "errorCode is required",
		},
		{
			name: "missing normalValue",
			config: HealthConfig{
				Checks: []HealthCheck{
					{
						Id:          "temp-check",
						DisplayName: "Temperature Check",
						Description: "Monitors temperature",
						ErrorCode:   "TEMP_ERROR",
						ValueSource: ValueSource{
							NodeId: "ns=2;s=Temp",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "normalValue is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && len(err.Error()) > 0 && len(tt.errMsg) > 0 {
					// Check if error message contains expected substring
					contains := false
					for i := 0; i <= len(err.Error())-len(tt.errMsg); i++ {
						if err.Error()[i:i+len(tt.errMsg)] == tt.errMsg {
							contains = true
							break
						}
					}
					if !contains {
						t.Errorf("HealthConfig.Validate() error = %v, expected to contain %v", err, tt.errMsg)
					}
				}
			}
		})
	}
}

func TestHealthConfig_Validate_PreservesNormalValue(t *testing.T) {
	normalValue := 42.5
	config := HealthConfig{
		Checks: []HealthCheck{
			{
				Id:          "temp-check",
				DisplayName: "Temperature Check",
				Description: "Monitors temperature",
				ErrorCode:   "TEMP_ERROR",
				NormalValue: &normalValue,
				ValueSource: ValueSource{
					NodeId: "ns=2;s=Temp",
				},
			},
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that normalValue is preserved
	if config.Checks[0].NormalValue == nil {
		t.Errorf("NormalValue should not be nil after Validate()")
	} else if *config.Checks[0].NormalValue != 42.5 {
		t.Errorf("NormalValue should be preserved as 42.5, got %v", *config.Checks[0].NormalValue)
	}
}
