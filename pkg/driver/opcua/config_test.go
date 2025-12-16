package opcua

import (
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
)

func TestReadBytes_ValidatesTraitConfigs(t *testing.T) {
	tests := []struct {
		name       string
		configJSON string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid config with Meter trait",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Tag1"}],
					"traits": [{
						"kind": "smartcore.bos.Meter",
						"unit": "kWh",
						"usage": {"nodeId": "ns=2;s=Tag1"}
					}]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "invalid config - Meter missing usage",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Tag1"}],
					"traits": [{
						"kind": "smartcore.bos.Meter",
						"unit": "kWh"
					}]
				}]
			}`,
			wantErr: true,
			errMsg:  "meter trait: usage is required",
		},
		{
			name: "invalid config - nodeId not in device variables",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Tag1"}],
					"traits": [{
						"kind": "smartcore.bos.Meter",
						"unit": "kWh",
						"usage": {"nodeId": "ns=2;s=Tag99"}
					}]
				}]
			}`,
			wantErr: true,
			errMsg:  "references nodeId 'ns=2;s=Tag99' which is not in device variables list",
		},
		{
			name: "invalid config - Electric trait missing demand",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Tag1"}],
					"traits": [{
						"kind": "smartcore.traits.Electric"
					}]
				}]
			}`,
			wantErr: true,
			errMsg:  "electric trait: demand is required",
		},
		{
			name: "invalid config - Transport trait no fields",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Tag1"}],
					"traits": [{
						"kind": "smartcore.bos.Transport"
					}]
				}]
			}`,
			wantErr: true,
			errMsg:  "transport trait: at least one field must be configured",
		},
		{
			name: "invalid config - udmi trait no points",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Tag1"}],
					"traits": [{
						"kind": "smartcore.bos.UDMI",
						"topicPrefix": "test/"
					}]
				}]
			}`,
			wantErr: true,
			errMsg:  "udmi trait: at least one point must be configured",
		},
		{
			name: "valid config with Electric trait",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=RealPower"}],
					"traits": [{
						"kind": "smartcore.traits.Electric",
						"demand": {
							"realPower": {"nodeId": "ns=2;s=RealPower"}
						}
					}]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "valid config with Transport trait",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Position"}],
					"traits": [{
						"kind": "smartcore.bos.Transport",
						"actualPosition": {"nodeId": "ns=2;s=Position"}
					}]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "valid config with udmi trait",
			configJSON: `{
				"name": "test-opcua",
				"conn": {
					"endpoint": "opc.tcp://localhost:4840"
				},
				"devices": [{
					"name": "test-device",
					"variables": [{"nodeId": "ns=2;s=Tag1"}],
					"traits": [{
						"kind": "smartcore.bos.UDMI",
						"topicPrefix": "test/",
						"points": {
							"point1": {"nodeId": "ns=2;s=Tag1"}
						}
					}]
				}]
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.ParseConfig([]byte(tt.configJSON))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("ParseConfig() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
