package protopkg

import (
	"testing"
)

type pathPair struct {
	name string
	old  string
	new  string
}

// pathPairs defines test cases with both old and new format paths
var pathPairs = []pathPair{
	{"MeterApi", "/smartcore.bos.MeterApi/GetMeterReading", "/smartcore.bos.meter.v1.MeterApi/GetMeterReading"},
	{"MeterInfo", "/smartcore.bos.MeterInfo/DescribeMeterReading", "/smartcore.bos.meter.v1.MeterInfo/DescribeMeterReading"},
	{"MeterHistory", "/smartcore.bos.MeterHistory/ListMeterReadingHistory", "/smartcore.bos.meter.v1.MeterHistory/ListMeterReadingHistory"},
	{"AlertApi", "/smartcore.bos.AlertApi/ListAlerts", "/smartcore.bos.alert.v1.AlertApi/ListAlerts"},
	{"AlertAdminApi", "/smartcore.bos.AlertAdminApi/CreateAlert", "/smartcore.bos.alert.v1.AlertAdminApi/CreateAlert"},
	{"ElectricHistory", "/smartcore.bos.ElectricHistory/ListElectricDemandHistory", "/smartcore.bos.electric.v1.ElectricHistory/ListElectricDemandHistory"},
}

func TestNewToOld(t *testing.T) {
	tests := append([]pathPair{
		{"old format unchanged", "/smartcore.bos.MeterApi/GetMeterReading", "/smartcore.bos.MeterApi/GetMeterReading"},
		{"different package unchanged", "/other.package.Service/Method", "/other.package.Service/Method"},
	}, pathPairs...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newToOld(tt.new)
			if result != tt.old {
				t.Errorf("got %q, want %q", result, tt.old)
			}
		})
	}
}

func TestOldToNew(t *testing.T) {
	tests := append([]pathPair{
		{"new format unchanged", "/smartcore.bos.meter.v1.MeterApi/GetMeterReading", "/smartcore.bos.meter.v1.MeterApi/GetMeterReading"},
		{"unrelated path unchanged", "/other.package.Service/Method", "/other.package.Service/Method"},
	}, pathPairs...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := oldToNew(tt.old)
			if result != tt.new {
				t.Errorf("got %q, want %q", result, tt.new)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	for _, tt := range pathPairs {
		t.Run(tt.name, func(t *testing.T) {
			// Old -> New -> Old
			newPath := oldToNew(tt.old)
			backToOld := newToOld(newPath)
			if backToOld != tt.old {
				t.Errorf("round trip failed: %q -> %q -> %q", tt.old, newPath, backToOld)
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
