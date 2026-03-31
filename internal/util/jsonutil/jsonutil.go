package jsonutil

import (
	"encoding/json"
	"maps"
)

// MarshalObjects marshals multiple objects into a single JSON object.
// If the same field is present in multiple objects, the last one wins.
// If any input does not marshal to a JSON object, an error is returned.
// Whether a field is considered present depends on if it appears in the output of json.Marshal -
// therefore struct field without 'omitempty' tag will always be considered present.
func MarshalObjects(objects ...any) ([]byte, error) {
	result := make(map[string]json.RawMessage)
	for _, obj := range objects {
		data, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		maps.Copy(result, m)
	}
	return json.Marshal(result)
}
