package udmi

import "time"

// PointsetVersion is the UDMI schema version stamped into pointset events.
const PointsetVersion = "1.5.2"

// PointsetEvent is the JSON payload of a UDMI pointset event message: the
// {timestamp, version, points} envelope required by events_pointset.json.
// see https://faucetsdn.github.io/udmi/docs/messages/pointset.html#event
type PointsetEvent struct {
	Timestamp time.Time   `json:"timestamp"`
	Version   string      `json:"version"`
	Points    PointsEvent `json:"points"`
}

// PointsEvent presents the JSON payload of a UDMI Event message
// see https://faucetsdn.github.io/udmi/docs/messages/pointset.html#event
type PointsEvent map[string]PointValue

func (f PointsEvent) Equal(other PointsEvent) bool {
	if f == nil && other == nil {
		return true
	}
	if f == nil || other == nil {
		return false
	}
	if len(f) != len(other) {
		return false
	}
	for key, value := range f {
		v, ok := other[key]
		if !ok {
			return false
		}
		if value == v {
			continue
		}
		if value.PresentValue != v.PresentValue {
			return false
		}
	}
	return true
}

// PointValue is a single UDMI point value
// see https://faucetsdn.github.io/udmi/docs/messages/pointset.html#event
type PointValue struct {
	// should be a primitive value: string, bool, float...
	PresentValue any `json:"present_value"`
}

// StateEvent is the JSON payload of a UDMI state message (state.json), which
// reports a device's current operational status.
// see https://faucetsdn.github.io/udmi/docs/messages/state.html
type StateEvent struct {
	Timestamp time.Time   `json:"timestamp"`
	Version   string      `json:"version"`
	System    StateSystem `json:"system"`
}

// StateSystem is the system block of a state message. state_system.json requires
// all of serial_no, last_config, hardware, software and operation.
type StateSystem struct {
	SerialNo   string         `json:"serial_no"`
	LastConfig time.Time      `json:"last_config"`
	Hardware   SystemHardware `json:"hardware"`
	// Software keys must match ^[a-z_]+$ (e.g. "firmware", "driver").
	Software  map[string]string `json:"software"`
	Operation SystemOperation   `json:"operation"`
}

// SystemHardware identifies the physical device; make and model are required.
type SystemHardware struct {
	Make  string `json:"make"`
	Model string `json:"model"`
}

// SystemOperation carries the device's operational status; operational is required.
type SystemOperation struct {
	Operational bool `json:"operational"`
}

// MetadataEvent is the JSON payload of a UDMI metadata message (metadata.json),
// the device's declared model.
// see https://faucetsdn.github.io/udmi/docs/messages/metadata.html
type MetadataEvent struct {
	Timestamp time.Time      `json:"timestamp"`
	Version   string         `json:"version"`
	System    MetadataSystem `json:"system"`
}

// MetadataSystem is the system block of a metadata message (model_system.json);
// all fields are optional, but a present location must carry a site.
type MetadataSystem struct {
	Name     string            `json:"name,omitempty"`
	Location *MetadataLocation `json:"location,omitempty"`
}

// MetadataLocation describes where the device is installed; site is required
// when location is present and must match ^[A-Z]{2}-[A-Z]{3,4}-[A-Z0-9]{2,9}$.
type MetadataLocation struct {
	Site string `json:"site"`
}

// ConfigMessage is a UDMI config message, used for control/settings
// https://faucetsdn.github.io/udmi/docs/messages/pointset.html#config
type ConfigMessage struct {
	PointSet ConfigPointSet `json:"pointset"`
}

// ConfigPointSet is a UDMI point set, for config messages
// https://faucetsdn.github.io/udmi/docs/messages/pointset.html#config
type ConfigPointSet struct {
	Points map[string]PointSetValue `json:"points"`
}

// PointSetValue is a single UDMI point with set value, for config messages
// https://faucetsdn.github.io/udmi/docs/messages/pointset.html#config
type PointSetValue struct {
	SetValue any `json:"set_value"`
}
