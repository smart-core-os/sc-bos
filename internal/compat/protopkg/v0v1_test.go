package protopkg

import (
	"testing"
)

// pkgPairs defines test cases with both v0 and v1 package formats for bidirectional translation.
var pkgPairs = []struct {
	name string
	v0   string
	v1   string
}{
	// Standard smartcore.bos services
	{"MeterApi", "smartcore.bos.MeterApi", "smartcore.bos.meter.v1.MeterApi"},
	{"AlertAdminApi", "smartcore.bos.AlertAdminApi", "smartcore.bos.alert.v1.AlertAdminApi"},
	{"MeterHistory", "smartcore.bos.MeterHistory", "smartcore.bos.meter.v1.MeterHistory"},
	{"AccountApi", "smartcore.bos.AccountApi", "smartcore.bos.account.v1.AccountApi"},
	{"UdmiService", "smartcore.bos.UdmiService", "smartcore.bos.udmi.v1.UdmiService"},
	{"MqttService", "smartcore.bos.MqttService", "smartcore.bos.mqtt.v1.MqttService"},
	// Nested packages
	{"driver.dali", "smartcore.bos.driver.dali.DaliApi", "smartcore.bos.driver.dali.v1.DaliApi"},
	{"driver.axiomxa", "smartcore.bos.driver.axiomxa.AxiomXaDriverService", "smartcore.bos.driver.axiomxa.v1.AxiomXaDriverService"},
	{"driver.bacnet", "smartcore.bos.driver.bacnet.BacnetDriverService", "smartcore.bos.driver.bacnet.v1.BacnetDriverService"},
	// Special case packages
	{"tenants", "smartcore.bos.tenants.TenantApi", "smartcore.bos.tenant.v1.TenantApi"},
	{"enterleavesensor", "smartcore.bos.EnterLeaveHistory", "smartcore.bos.enterleavesensor.v1.EnterLeaveSensorHistory"},
	// Packages that should remain unchanged (v0 == v1)
	{"v2 standard", "smartcore.bos.meter.v2.MeterApi", "smartcore.bos.meter.v2.MeterApi"},
	{"v2 nested", "smartcore.bos.driver.dali.v2.DaliApi", "smartcore.bos.driver.dali.v2.DaliApi"},
	{"non-smartcore", "other.package.SomeApi", "other.package.SomeApi"},
	{"almost smartcore", "smartcore.bosx.SomeApi", "smartcore.bosx.SomeApi"},
}

func TestV0ToV1(t *testing.T) {
	for _, tt := range pkgPairs {
		t.Run(tt.name, func(t *testing.T) {
			result := V0ToV1(tt.v0)
			if result != tt.v1 {
				t.Errorf("V0ToV1(%q) = %q, want %q", tt.v0, result, tt.v1)
			}

			// Test idempotency: v1 input should return v1 unchanged
			idempotent := V0ToV1(tt.v1)
			if idempotent != tt.v1 {
				t.Errorf("V0ToV1(%q) not idempotent: got %q, want %q", tt.v1, idempotent, tt.v1)
			}
		})
	}
}

func TestV1ToV0(t *testing.T) {
	for _, tt := range pkgPairs {
		t.Run(tt.name, func(t *testing.T) {
			result := V1ToV0(tt.v1)
			if result != tt.v0 {
				t.Errorf("V1ToV0(%q) = %q, want %q", tt.v1, result, tt.v0)
			}

			// Test idempotency: v0 input should return v0 unchanged
			idempotent := V1ToV0(tt.v0)
			if idempotent != tt.v0 {
				t.Errorf("V1ToV0(%q) not idempotent: got %q, want %q", tt.v0, idempotent, tt.v0)
			}
		})
	}
}

// traitPairs defines test cases for bidirectional translation between smartcore.traits and v1 formats.
var traitPairs = []struct {
	name   string
	traits string
	v1     string
}{
	{"MeterApi", "smartcore.traits.MeterApi", "smartcore.bos.meter.v1.MeterApi"},
	{"MeterInfo", "smartcore.traits.MeterInfo", "smartcore.bos.meter.v1.MeterInfo"},
	{"EnterLeaveSensorApi", "smartcore.traits.EnterLeaveSensorApi", "smartcore.bos.enterleavesensor.v1.EnterLeaveSensorApi"},
	{"EnterLeaveSensorInfo", "smartcore.traits.EnterLeaveSensorInfo", "smartcore.bos.enterleavesensor.v1.EnterLeaveSensorInfo"},
	// Special case: MotionSensorSensorInfo is a typo corrected to MotionSensorInfo
	{"MotionSensorSensorInfo", "smartcore.traits.MotionSensorSensorInfo", "smartcore.bos.motionsensor.v1.MotionSensorInfo"},
	{"MotionSensorApi", "smartcore.traits.MotionSensorApi", "smartcore.bos.motionsensor.v1.MotionSensorApi"},
}

func TestTraitsToV1(t *testing.T) {
	for _, tt := range traitPairs {
		t.Run(tt.name, func(t *testing.T) {
			result := TraitsToV1(tt.traits)
			if result != tt.v1 {
				t.Errorf("TraitsToV1(%q) = %q, want %q", tt.traits, result, tt.v1)
			}
		})
	}

	// Non-traits packages should be unchanged
	t.Run("non-traits unchanged", func(t *testing.T) {
		input := "other.package.SomeApi"
		if got := TraitsToV1(input); got != input {
			t.Errorf("TraitsToV1(%q) = %q, want unchanged", input, got)
		}
	})

	// smartcore.bos packages should be unchanged
	t.Run("smartcore.bos unchanged", func(t *testing.T) {
		input := "smartcore.bos.MeterApi"
		if got := TraitsToV1(input); got != input {
			t.Errorf("TraitsToV1(%q) = %q, want unchanged", input, got)
		}
	})
}

func TestV1ToTraits(t *testing.T) {
	for _, tt := range traitPairs {
		t.Run(tt.name, func(t *testing.T) {
			result := V1ToTraits(tt.v1)
			if result != tt.traits {
				t.Errorf("V1ToTraits(%q) = %q, want %q", tt.v1, result, tt.traits)
			}
		})
	}

	// Nested packages should be unchanged
	t.Run("nested package unchanged", func(t *testing.T) {
		input := "smartcore.bos.driver.dali.v1.DaliApi"
		if got := V1ToTraits(input); got != input {
			t.Errorf("V1ToTraits(%q) = %q, want unchanged", input, got)
		}
	})

	// Non-bos packages should be unchanged
	t.Run("non-bos unchanged", func(t *testing.T) {
		input := "other.package.SomeApi"
		if got := V1ToTraits(input); got != input {
			t.Errorf("V1ToTraits(%q) = %q, want unchanged", input, got)
		}
	})
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
