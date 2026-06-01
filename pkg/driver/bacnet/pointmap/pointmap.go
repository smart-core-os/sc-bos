// Package pointmap derives the per-device "point map" advertised in Smart
// Core's metadatapb.Metadata.More for BACnet devices.
//
// The point map is built directly from the driver's existing UDMI trait
// configuration (kind: udmi) - the same config that drives MQTT pointset
// publishing - so there is no second list to hand-maintain. Each MQTT point in
// a UDMI trait whose value source resolves to a given device becomes one Point
// in that device's point_map.
//
// The Point struct mirrors the cross-integration shape used elsewhere
// so consumers parse every integration's point_map uniformly.
//
// Fields beyond mqtt/spec/access come from the driver's PointSpecs config
// (config.Root.PointSpecs, keyed by MQTT point name). When a point has no
// matching PointSpec entry: "spec" defaults to the MQTT name, "meaning",
// "type" and "unit" are left empty, and "access" is inferred from the BACnet
// object type backing the point (Output/Value -> "read/write"; refs by name
// fall back to "read").
package pointmap

import (
	"encoding/json"
	"path"
	"sort"

	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

// Point is one published point: the spec name a consumer references it by, the
// MQTT point name the consumer receives, and how to interpret it. Field names
// are stable so consumers can parse point_map identically across integrations.
type Point struct {
	Spec    string `json:"spec"`
	Mqtt    string `json:"mqtt"`
	Meaning string `json:"meaning"`
	Type    string `json:"type"`
	Unit    string `json:"unit,omitempty"`
	Access  string `json:"access"`
}

const (
	accessRead      = "read"
	accessReadWrite = "read/write"
)

// JSON marshals points to a JSON array - the value stored in
// Metadata.More["point_map"]. Returns "" when points is empty so callers can
// omit the More entry rather than emit "[]".
func JSON(points []Point) string {
	if len(points) == 0 {
		return ""
	}
	b, err := json.Marshal(points)
	if err != nil {
		return ""
	}
	return string(b)
}

// More builds the metadatapb Metadata.More map for a device: its MQTT topic and
// its point catalogue. mqttTopic is "<mqttTopicPrefix>/<deviceId>" with no
// suffix; consumers append "/events/pointset" etc themselves.
func More(mqttTopic string, points []Point) map[string]string {
	m := make(map[string]string, 2)
	if mqttTopic != "" {
		m["mqtt_topic"] = mqttTopic
	}
	if pm := JSON(points); pm != "" {
		m["point_map"] = pm
	}
	return m
}

// TopicFor returns the base MQTT topic for a device: "<prefix>/<deviceName>".
// Returns "" when prefix or deviceName is empty.
func TopicFor(mqttTopicPrefix, deviceName string) string {
	if mqttTopicPrefix == "" || deviceName == "" {
		return ""
	}
	return path.Join(mqttTopicPrefix, deviceName)
}

// ForDevice walks the configured traits and returns the Point list for the
// named device - one Point per MQTT point in any UDMI trait whose value source
// resolves to this device. Output is sorted by mqtt name for deterministic
// metadata.
func ForDevice(cfg config.Root, device config.Device) []Point {
	var out []Point
	seen := make(map[string]struct{})
	for _, t := range cfg.Traits {
		if string(t.Kind) != "udmi" {
			continue
		}
		points, err := pointsFromUDMIRaw(t.Raw)
		if err != nil {
			continue
		}
		for mqtt, vs := range points {
			if vs == nil || vs.Device == nil {
				continue
			}
			if !deviceMatches(*vs.Device, device) {
				continue
			}
			if _, dup := seen[mqtt]; dup {
				continue
			}
			seen[mqtt] = struct{}{}
			out = append(out, buildPoint(mqtt, vs, cfg.PointSpecs[mqtt]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mqtt < out[j].Mqtt })
	return out
}

// buildPoint composes one Point, layering operator-supplied PointSpec values on
// top of the defaults derived from the value source.
func buildPoint(mqtt string, vs *config.ValueSource, spec config.PointSpec) Point {
	p := Point{
		Spec:    spec.Spec,
		Mqtt:    mqtt,
		Meaning: spec.Meaning,
		Type:    spec.Type,
		Unit:    spec.Unit,
		Access:  spec.Access,
	}
	if p.Spec == "" {
		p.Spec = mqtt
	}
	if p.Access == "" {
		p.Access = accessForObject(vs.Object)
	}
	return p
}

// pointsFromUDMIRaw decodes the bits of a UDMI trait we care about (the points
// map). We decode locally rather than importing merge.UdmiMergeConfig because
// merge imports config and this package would otherwise sit in a cycle.
func pointsFromUDMIRaw(raw []byte) (map[string]*config.ValueSource, error) {
	var u struct {
		Points map[string]*config.ValueSource `json:"points"`
	}
	if err := json.Unmarshal(raw, &u); err != nil {
		return nil, err
	}
	return u.Points, nil
}

func deviceMatches(ref config.DeviceRef, dev config.Device) bool {
	if n := ref.Name(); n != "" {
		return n == dev.Name
	}
	if id := ref.InstanceID(); id != 0 {
		return id == dev.ID
	}
	return false
}

// accessForObject infers read vs read/write from the BACnet object type, when
// known. Refs by name return "read" because the type isn't determinable
// without resolving against a live device.
func accessForObject(ref *config.ObjectRef) string {
	if ref == nil {
		return accessRead
	}
	id := ref.ID()
	if id == (config.ObjectID{}) {
		return accessRead
	}
	switch bactypes.ObjectID(id).Type {
	case objecttype.AnalogOutput, objecttype.BinaryOutput, objecttype.MultiStateOutput,
		objecttype.AnalogValue, objecttype.BinaryValue, objecttype.MultiStateValue:
		return accessReadWrite
	}
	return accessRead
}
