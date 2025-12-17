package protopkg

import (
	"testing"
)

// pkgPairs defines test cases with both v0 and v1 package formats for bidirectional translation.
var pkgPairs = []struct {
	name    string
	v0      string
	v1      string
	service string
}{
	// Standard smartcore.bos services
	{"MeterApi", "smartcore.bos", "smartcore.bos.meter.v1", "MeterApi"},
	{"AlertAdminApi", "smartcore.bos", "smartcore.bos.alert.v1", "AlertAdminApi"},
	{"MeterHistory", "smartcore.bos", "smartcore.bos.meter.v1", "MeterHistory"},
	{"AccountApi", "smartcore.bos", "smartcore.bos.account.v1", "AccountApi"},
	// Nested packages
	{"driver.dali", "smartcore.bos.driver.dali", "smartcore.bos.driver.dali.v1", "DaliApi"},
	{"driver.axiomxa", "smartcore.bos.driver.axiomxa", "smartcore.bos.driver.axiomxa.v1", "AxiomXaDriverService"},
	{"driver.bacnet", "smartcore.bos.driver.bacnet", "smartcore.bos.driver.bacnet.v1", "BacnetDriverService"},
	// Special case packages
	{"tenants", "smartcore.bos.tenants", "smartcore.bos.tenants.v1", "TenantApi"},
	{"udmi", "smartcore.bos", "smartcore.bos.udmi.v1", "UdmiService"},
	{"mqtt", "smartcore.bos", "smartcore.bos.mqtt.v1", "MqttService"},
	// Packages that should remain unchanged (v0 == v1)
	{"v2 standard", "smartcore.bos.meter.v2", "smartcore.bos.meter.v2", "MeterApi"},
	{"v2 nested", "smartcore.bos.driver.dali.v2", "smartcore.bos.driver.dali.v2", "DaliApi"},
	{"non-smartcore", "other.package", "other.package", "SomeApi"},
	{"almost smartcore", "smartcore.bosx", "smartcore.bosx", "SomeApi"},
}

func TestV0ToV1(t *testing.T) {
	for _, tt := range pkgPairs {
		t.Run(tt.name, func(t *testing.T) {
			result := V0ToV1(tt.v0, tt.service)
			if result != tt.v1 {
				t.Errorf("V0ToV1(%q, %q) = %q, want %q", tt.v0, tt.service, result, tt.v1)
			}

			// Test idempotency: v1 input should return v1 unchanged
			idempotent := V0ToV1(tt.v1, tt.service)
			if idempotent != tt.v1 {
				t.Errorf("V0ToV1(%q, %q) not idempotent: got %q, want %q", tt.v1, tt.service, idempotent, tt.v1)
			}
		})
	}
}

func TestV1ToV0(t *testing.T) {
	for _, tt := range pkgPairs {
		t.Run(tt.name, func(t *testing.T) {
			result := V1ToV0(tt.v1, tt.service)
			if result != tt.v0 {
				t.Errorf("V1ToV0(%q, %q) = %q, want %q", tt.v1, tt.service, result, tt.v0)
			}

			// Test idempotency: v0 input should return v0 unchanged
			idempotent := V1ToV0(tt.v0, tt.service)
			if idempotent != tt.v0 {
				t.Errorf("V1ToV0(%q, %q) not idempotent: got %q, want %q", tt.v0, tt.service, idempotent, tt.v0)
			}
		})
	}
}

func TestExtractResource(t *testing.T) {
	tests := []struct {
		service  string
		expected string
	}{
		{"MeterApi", "meter"},
		{"MeterInfo", "meter"},
		{"MeterHistory", "meter"},
		{"AlertApi", "alert"},
		{"AlertAdminApi", "alert"},
		{"ElectricHistory", "electric"},
		{"TemperatureApi", "temperature"},
		{"OccupancySensorHistory", "occupancysensor"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			result := extractResource(tt.service)
			if result != tt.expected {
				t.Errorf("extractResource(%q) = %q, want %q", tt.service, result, tt.expected)
			}
		})
	}
}
