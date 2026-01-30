package config

import (
	"testing"
)

func TestValueSource_Scaled(t *testing.T) {
	tests := []struct {
		name     string
		source   ValueSource
		input    any
		expected any
	}{
		{
			name:     "no scale - float32",
			source:   ValueSource{Scale: 0},
			input:    float32(10.5),
			expected: float32(10.5),
		},
		{
			name:     "scale 1 - float32",
			source:   ValueSource{Scale: 1},
			input:    float32(10.5),
			expected: float32(10.5),
		},
		{
			name:     "scale 1000 - float32 (kW to W)",
			source:   ValueSource{Scale: 1000},
			input:    float32(2.5),
			expected: float32(2500),
		},
		{
			name:     "scale 0.001 - float32 (W to kW)",
			source:   ValueSource{Scale: 0.001},
			input:    float32(2500),
			expected: float32(2.5),
		},
		{
			name:     "scale 1000 - float64",
			source:   ValueSource{Scale: 1000},
			input:    float64(2.5),
			expected: float64(2500),
		},
		{
			name:     "scale 1000 - int32",
			source:   ValueSource{Scale: 1000},
			input:    int32(2),
			expected: int32(2000),
		},
		{
			name:     "scale 1000 - uint32",
			source:   ValueSource{Scale: 1000},
			input:    uint32(2),
			expected: uint32(2000),
		},
		{
			name:     "nil value",
			source:   ValueSource{Scale: 1000},
			input:    nil,
			expected: nil,
		},
		{
			name:     "string value (no scaling)",
			source:   ValueSource{Scale: 1000},
			input:    "test",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.source.Scaled(tt.input)
			if result != tt.expected {
				t.Errorf("Scaled() = %v (type %T), expected %v (type %T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestValueSource_Scaled_Precision(t *testing.T) {
	// Test that float32 precision is maintained for typical kW to W conversions
	source := ValueSource{Scale: 1000}

	tests := []struct {
		input    float32
		expected float32
	}{
		{0.001, 1.0},
		{1.5, 1500.0},
		{10.25, 10250.0},
		{0.0, 0.0},
	}

	for _, tt := range tests {
		result := source.Scaled(tt.input)
		resultFloat, ok := result.(float32)
		if !ok {
			t.Errorf("Scaled(%v) returned type %T, expected float32", tt.input, result)
			continue
		}
		if resultFloat != tt.expected {
			t.Errorf("Scaled(%v) = %v, expected %v", tt.input, resultFloat, tt.expected)
		}
	}
}
