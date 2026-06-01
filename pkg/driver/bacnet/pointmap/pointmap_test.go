package pointmap

import (
	"encoding/json"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

// sampleRoot is a minimal bacnet config with one device and one UDMI trait that
// publishes two MQTT points referencing it. One has an operator-supplied
// PointSpec, the other doesn't so the defaults apply.
func sampleRoot(t *testing.T) (config.Root, config.Device) {
	t.Helper()
	const traitJSON = `{
		"name": "ahu-1-udmi",
		"kind": "udmi",
		"topicPrefix": "ACME/BUILDING/AHU-1",
		"points": {
			"supply-air-temp-c":  {"device": "ahu-1", "object": "AnalogInput:0"},
			"supply-fan-speed":   {"device": "ahu-1", "object": "AnalogOutput:1"}
		}
	}`
	rt := config.RawTrait{}
	if err := json.Unmarshal([]byte(traitJSON), &rt); err != nil {
		t.Fatalf("unmarshal trait: %v", err)
	}

	dev := config.Device{Name: "ahu-1", ID: 10000, Title: "AHU-1 Vendor"}
	root := config.Root{
		MqttTopicPrefix: "ACME/BUILDING",
		PointSpecs: map[string]config.PointSpec{
			"supply-air-temp-c": {Spec: "SAT", Meaning: "supply.air.temperature", Type: "real", Unit: "C", Access: "read"},
		},
		Devices: []config.Device{dev},
		Traits:  []config.RawTrait{rt},
	}
	// Trait.Kind comes from the embedded Trait struct, populated by RawTrait
	// UnmarshalJSON; explicitly check it picked it up.
	if root.Traits[0].Kind != trait.Name("udmi") {
		t.Fatalf("trait kind = %q, want udmi", root.Traits[0].Kind)
	}
	return root, dev
}

func TestForDevice_BuildsPointsWithSpecOverridesAndDefaults(t *testing.T) {
	root, dev := sampleRoot(t)
	points := ForDevice(root, dev)
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2: %+v", len(points), points)
	}

	// Output is sorted by mqtt name. Index 0 is supply-air-temp-c.
	sat := points[0]
	if sat.Mqtt != "supply-air-temp-c" {
		t.Fatalf("points[0].Mqtt = %q", sat.Mqtt)
	}
	if sat.Spec != "SAT" || sat.Meaning != "supply.air.temperature" || sat.Type != "real" || sat.Unit != "C" || sat.Access != "read" {
		t.Errorf("PointSpec did not propagate: %+v", sat)
	}

	// Index 1 has no PointSpec, so spec defaults to mqtt and access is inferred
	// from the AnalogOutput type (read/write).
	fan := points[1]
	if fan.Mqtt != "supply-fan-speed" {
		t.Fatalf("points[1].Mqtt = %q", fan.Mqtt)
	}
	if fan.Spec != fan.Mqtt {
		t.Errorf("spec should default to mqtt, got %q", fan.Spec)
	}
	if fan.Access != "read/write" {
		t.Errorf("AnalogOutput should infer read/write, got %q", fan.Access)
	}
	if fan.Meaning != "" || fan.Type != "" || fan.Unit != "" {
		t.Errorf("unspecified fields should be empty: %+v", fan)
	}
}

func TestMoreEmitsTopicAndValidJSON(t *testing.T) {
	root, dev := sampleRoot(t)
	points := ForDevice(root, dev)
	topic := TopicFor(root.MqttTopicPrefix, dev.Name)
	m := More(topic, points)

	if m["mqtt_topic"] != "ACME/BUILDING/ahu-1" {
		t.Errorf("mqtt_topic = %q", m["mqtt_topic"])
	}
	if m["point_map"] == "" {
		t.Fatal("point_map missing")
	}
	if !json.Valid([]byte(m["point_map"])) {
		t.Fatalf("point_map is not valid JSON: %s", m["point_map"])
	}

	// Round-trip the JSON to confirm field names are exactly the documented ones.
	var got []Point
	if err := json.Unmarshal([]byte(m["point_map"]), &got); err != nil {
		t.Fatalf("unmarshal point_map: %v", err)
	}
	if len(got) != len(points) {
		t.Fatalf("round-trip length mismatch: got %d want %d", len(got), len(points))
	}
}

func TestForDevice_NoUDMITraitsReturnsEmpty(t *testing.T) {
	root := config.Root{MqttTopicPrefix: "ACME/BUILDING"}
	if pts := ForDevice(root, config.Device{Name: "ahu-1", ID: 10000}); len(pts) != 0 {
		t.Errorf("expected no points, got %+v", pts)
	}
}

func TestMoreOmitsEmptyEntries(t *testing.T) {
	m := More("", nil)
	if _, ok := m["mqtt_topic"]; ok {
		t.Error("mqtt_topic should be omitted when topic is empty")
	}
	if _, ok := m["point_map"]; ok {
		t.Error("point_map should be omitted when there are no points")
	}
}
