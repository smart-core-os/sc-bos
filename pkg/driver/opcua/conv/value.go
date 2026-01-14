package conv

import (
	"fmt"
	"strconv"

	"golang.org/x/exp/constraints"
)

func IntValue(data any) (int, error) {
	switch v := data.(type) {
	case error:
		return 0, v
	case int32:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	}

	return 0, fmt.Errorf("unsupported conversion %T -> int32 for val %v", data, data)
}

func Float32Value(data any) (float32, error) {
	switch v := data.(type) {
	case error:
		return 0, v
	case float32:
		return v, nil
	case uint8:
		return float32(v), nil
	case uint16:
		return float32(v), nil
	case uint32:
		return float32(v), nil
	case int8:
		return float32(v), nil
	case int16:
		return float32(v), nil
	case int32:
		return float32(v), nil
	}

	return 0, fmt.Errorf("unsupported conversion %T -> float32 for val %v", data, data)
}

func Float64Value(data any) (float64, error) {
	switch v := data.(type) {
	case float64:
		return v, nil
	default:
		val, err := Float32Value(data)
		return float64(val), err
	}
}

// ToString converts a value to its string representation
func ToString(data any) (string, error) {
	switch v := data.(type) {
	case error:
		return v.Error(), nil
	case string:
		return v, nil
	}

	if v, err := IntValue(data); err == nil {
		return strconv.Itoa(v), nil
	} else if v, err := Float32Value(data); err == nil {
		return strconv.FormatFloat(float64(v), 'f', 2, 32), nil
	}

	return "", fmt.Errorf("unsupported conversion %T -> string for val %v", data, data)
}

// ToTraitEnum takes the value read from the OPC UA node and looks up the value in the enum map defined in the config.
// If value is found, this value is then used to look up the enum value in the proto pb <EnumName>_value field.
func ToTraitEnum[T constraints.Integer](data any, enum map[string]string, traitMap map[string]int32) (T, error) {
	if enum != nil {
		s, err := ToString(data)
		if err == nil {
			if v, ok := enum[s]; ok {
				if r, ok := traitMap[v]; ok {
					return T(r), nil
				}
			}
		}
	}
	return 0, fmt.Errorf("unsupported conversion %T -> enum for val %v", data, data)
}
