package protopkg

import (
	"testing"
)

func TestV0ToV1(t *testing.T) {
	tests := []struct {
		name        string
		pkg         string
		service     string
		expectedPkg string
	}{
		{"MeterApi", "smartcore.bos", "MeterApi", "smartcore.bos.meter.v1"},
		{"AlertAdminApi", "smartcore.bos", "AlertAdminApi", "smartcore.bos.alert.v1"},
		{"MeterHistory", "smartcore.bos", "MeterHistory", "smartcore.bos.meter.v1"},
		{"non-smartcore.bos", "other.package", "SomeApi", "other.package"},
		{"nested package", "smartcore.bos.driver.dali", "DaliApi", "smartcore.bos.driver.dali"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := V0ToV1(tt.pkg, tt.service)
			if result != tt.expectedPkg {
				t.Errorf("got %q, want %q", result, tt.expectedPkg)
			}
		})
	}
}

func TestV1ToV0(t *testing.T) {
	tests := []struct {
		name        string
		pkg         string
		service     string
		expectedPkg string
	}{
		{"MeterApi", "smartcore.bos.meter.v1", "MeterApi", "smartcore.bos"},
		{"AlertAdminApi", "smartcore.bos.alert.v1", "AlertAdminApi", "smartcore.bos"},
		{"MeterHistory", "smartcore.bos.meter.v1", "MeterHistory", "smartcore.bos"},
		{"non-v1", "smartcore.bos.meter.v2", "MeterApi", "smartcore.bos.meter.v2"},
		{"v0", "smartcore.bos", "MeterApi", "smartcore.bos"},
		{"nested package", "smartcore.bos.driver.dali.v1", "DaliApi", "smartcore.bos.driver.dali.v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := V1ToV0(tt.pkg, tt.service)
			if result != tt.expectedPkg {
				t.Errorf("got %q, want %q", result, tt.expectedPkg)
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
