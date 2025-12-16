package conv

import (
	"errors"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/gen"
)

func TestIntValue(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      int
		wantError bool
	}{
		{
			name:      "int32",
			input:     int32(100),
			want:      100,
			wantError: false,
		},
		{
			name:      "uint8",
			input:     uint8(50),
			want:      50,
			wantError: false,
		},
		{
			name:      "uint16",
			input:     uint16(1000),
			want:      1000,
			wantError: false,
		},
		{
			name:      "uint32",
			input:     uint32(50000),
			want:      50000,
			wantError: false,
		},
		{
			name:      "int8",
			input:     int8(-50),
			want:      -50,
			wantError: false,
		},
		{
			name:      "int16",
			input:     int16(-1000),
			want:      -1000,
			wantError: false,
		},
		{
			name:      "error input",
			input:     errors.New("test error"),
			want:      0,
			wantError: true,
		},
		{
			name:      "unsupported type - string",
			input:     "not a number",
			want:      0,
			wantError: true,
		},
		{
			name:      "unsupported type - float32",
			input:     float32(123.45),
			want:      0,
			wantError: true,
		},
		{
			name:      "unsupported type - bool",
			input:     true,
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IntValue(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("IntValue() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("IntValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloat32Value(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      float32
		wantError bool
	}{
		{
			name:      "float32",
			input:     float32(123.45),
			want:      123.45,
			wantError: false,
		},
		{
			name:      "uint8",
			input:     uint8(50),
			want:      50.0,
			wantError: false,
		},
		{
			name:      "uint16",
			input:     uint16(1000),
			want:      1000.0,
			wantError: false,
		},
		{
			name:      "uint32",
			input:     uint32(50000),
			want:      50000.0,
			wantError: false,
		},
		{
			name:      "int8",
			input:     int8(-50),
			want:      -50.0,
			wantError: false,
		},
		{
			name:      "int16",
			input:     int16(-1000),
			want:      -1000.0,
			wantError: false,
		},
		{
			name:      "int32",
			input:     int32(100),
			want:      100.0,
			wantError: false,
		},
		{
			name:      "error input",
			input:     errors.New("test error"),
			want:      0,
			wantError: true,
		},
		{
			name:      "unsupported type - string",
			input:     "not a number",
			want:      0,
			wantError: true,
		},
		{
			name:      "unsupported type - bool",
			input:     false,
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Float32Value(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Float32Value() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("Float32Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      string
		wantError bool
	}{
		{
			name:      "string",
			input:     "hello",
			want:      "hello",
			wantError: false,
		},
		{
			name:      "error - converts to error message",
			input:     errors.New("test error"),
			want:      "test error",
			wantError: false,
		},
		{
			name:      "int32 via IntValue",
			input:     int32(42),
			want:      "42",
			wantError: false,
		},
		{
			name:      "int8 via IntValue",
			input:     int8(-10),
			want:      "-10",
			wantError: false,
		},
		{
			name:      "uint16 via IntValue",
			input:     uint16(1000),
			want:      "1000",
			wantError: false,
		},
		{
			name:      "float32 via Float32Value",
			input:     float32(123.456),
			want:      "123.46",
			wantError: false,
		},
		{
			name:      "float32 small value",
			input:     float32(0.12),
			want:      "0.12",
			wantError: false,
		},
		{
			name:      "unsupported type - bool",
			input:     true,
			want:      "",
			wantError: true,
		},
		{
			name:      "unsupported type - struct",
			input:     struct{ name string }{name: "test"},
			want:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToString(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ToString() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToTraitEnum(t *testing.T) {
	enumMap := map[string]string{
		"0": "OPEN",
		"1": "OPENING",
		"2": "CLOSED",
		"3": "CLOSING",
	}

	tests := []struct {
		name      string
		input     any
		enumMap   map[string]string
		traitMap  map[string]int32
		want      gen.Transport_Door_DoorStatus
		wantError bool
	}{
		{
			name:      "valid string conversion",
			input:     "1",
			enumMap:   enumMap,
			traitMap:  gen.Transport_Door_DoorStatus_value,
			want:      gen.Transport_Door_OPENING,
			wantError: false,
		},
		{
			name:      "valid int32 conversion",
			input:     int32(2),
			enumMap:   enumMap,
			traitMap:  gen.Transport_Door_DoorStatus_value,
			want:      gen.Transport_Door_CLOSED,
			wantError: false,
		},
		{
			name:      "valid int8 conversion",
			input:     int8(3),
			enumMap:   enumMap,
			traitMap:  gen.Transport_Door_DoorStatus_value,
			want:      gen.Transport_Door_CLOSING,
			wantError: false,
		},
		{
			name:      "nil enum map",
			input:     "1",
			enumMap:   nil,
			traitMap:  gen.Transport_Door_DoorStatus_value,
			want:      0,
			wantError: true,
		},
		{
			name:      "value not in enum map",
			input:     "99",
			enumMap:   enumMap,
			traitMap:  gen.Transport_Door_DoorStatus_value,
			want:      0,
			wantError: true,
		},
		{
			name:      "value not in trait map",
			input:     "1",
			enumMap:   map[string]string{"1": "INVALID_VALUE"},
			traitMap:  gen.Transport_Door_DoorStatus_value,
			want:      0,
			wantError: true,
		},
		{
			name:      "unsupported input type",
			input:     true,
			enumMap:   enumMap,
			traitMap:  gen.Transport_Door_DoorStatus_value,
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToTraitEnum[gen.Transport_Door_DoorStatus](tt.input, tt.enumMap, tt.traitMap)
			if (err != nil) != tt.wantError {
				t.Errorf("ToTraitEnum() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("ToTraitEnum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToTraitEnum_DifferentTypes(t *testing.T) {
	// Test with Transport_Direction enum
	directionMap := map[string]string{
		"0": "STATIONARY",
		"1": "UP",
		"2": "DOWN",
	}

	direction, err := ToTraitEnum[gen.Transport_Direction](int32(1), directionMap, gen.Transport_Direction_value)
	if err != nil {
		t.Errorf("ToTraitEnum() for Direction failed: %v", err)
	}
	if direction != gen.Transport_UP {
		t.Errorf("ToTraitEnum() for Direction = %v, want UP", direction)
	}

	// Test with OperatingMode enum
	modeMap := map[string]string{
		"0": "NORMAL",
		"1": "EMERGENCY",
	}

	mode, err := ToTraitEnum[gen.Transport_OperatingMode](int32(0), modeMap, gen.Transport_OperatingMode_value)
	if err != nil {
		t.Errorf("ToTraitEnum() for OperatingMode failed: %v", err)
	}
	if mode != gen.Transport_NORMAL {
		t.Errorf("ToTraitEnum() for OperatingMode = %v, want NORMAL", mode)
	}
}

func TestIntValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{
			name:  "zero int32",
			input: int32(0),
			want:  0,
		},
		{
			name:  "max uint8",
			input: uint8(255),
			want:  255,
		},
		{
			name:  "max int8",
			input: int8(127),
			want:  127,
		},
		{
			name:  "min int8",
			input: int8(-128),
			want:  -128,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IntValue(tt.input)
			if err != nil {
				t.Errorf("IntValue() unexpected error = %v", err)
			}
			if got != tt.want {
				t.Errorf("IntValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloat32Value_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  float32
	}{
		{
			name:  "zero",
			input: float32(0.0),
			want:  0.0,
		},
		{
			name:  "negative",
			input: float32(-123.45),
			want:  -123.45,
		},
		{
			name:  "very small",
			input: float32(0.001),
			want:  0.001,
		},
		{
			name:  "zero from int32",
			input: int32(0),
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Float32Value(tt.input)
			if err != nil {
				t.Errorf("Float32Value() unexpected error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Float32Value() = %v, want %v", got, tt.want)
			}
		})
	}
}
