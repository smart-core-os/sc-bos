package merge

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
)

func Test_pointsToPointSet_sanitisesNaNAndInf(t *testing.T) {
	um := &udmiMerge{}
	points := udmi.PointsEvent{
		"nan":      udmi.PointValue{PresentValue: math.NaN()},
		"posInf":   udmi.PointValue{PresentValue: math.Inf(1)},
		"negInf":   udmi.PointValue{PresentValue: math.Inf(-1)},
		"nan32":    udmi.PointValue{PresentValue: float32(math.NaN())},
		"posInf32": udmi.PointValue{PresentValue: float32(math.Inf(1))},
		"negInf32": udmi.PointValue{PresentValue: float32(math.Inf(-1))},
		"ok":       udmi.PointValue{PresentValue: 42.0},
		"str":      udmi.PointValue{PresentValue: "hello"},
	}

	um.config.TopicPrefix = "topic"
	msg, err := um.pointsToPointSet(points)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]udmi.PointValue
	if err := json.Unmarshal([]byte(msg.Payload), &out); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if out["nan"].PresentValue != "NaN" {
		t.Errorf("expected nan to be \"NaN\", got %v", out["nan"].PresentValue)
	}
	if out["posInf"].PresentValue != "Infinity" {
		t.Errorf("expected posInf to be \"Infinity\", got %v", out["posInf"].PresentValue)
	}
	if out["negInf"].PresentValue != "-Infinity" {
		t.Errorf("expected negInf to be \"-Infinity\", got %v", out["negInf"].PresentValue)
	}
	if out["nan32"].PresentValue != "NaN" {
		t.Errorf("expected nan32 to be \"NaN\", got %v", out["nan32"].PresentValue)
	}
	if out["posInf32"].PresentValue != "Infinity" {
		t.Errorf("expected posInf32 to be \"Infinity\", got %v", out["posInf32"].PresentValue)
	}
	if out["negInf32"].PresentValue != "-Infinity" {
		t.Errorf("expected negInf32 to be \"-Infinity\", got %v", out["negInf32"].PresentValue)
	}
	if out["ok"].PresentValue != 42.0 {
		t.Errorf("expected ok to be 42.0, got %v", out["ok"].PresentValue)
	}
	if out["str"].PresentValue != "hello" {
		t.Errorf("expected str to be 'hello', got %v", out["str"].PresentValue)
	}
}

// On the UDMI-spec topic the payload is the {timestamp, version, points} envelope.
func Test_pointsToPointSet_specTopicWrapsEnvelope(t *testing.T) {
	um := &udmiMerge{}
	um.config.TopicPrefix = "topic"
	suffix := "/events/pointset"
	um.config.TopicSuffix = &suffix

	msg, err := um.pointsToPointSet(udmi.PointsEvent{"ok": {PresentValue: 42.0}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var event udmi.PointsetEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if event.Version != udmi.PointsetVersion {
		t.Errorf("expected version %q, got %q", udmi.PointsetVersion, event.Version)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected a non-zero timestamp")
	}
	if event.Points["ok"].PresentValue != 42.0 {
		t.Errorf("points not carried through: %+v", event.Points)
	}
}

// A configured udmiVersion overrides the default stamped into the envelope.
func Test_pointsToPointSet_versionConfigurable(t *testing.T) {
	um := &udmiMerge{}
	um.config.TopicPrefix = "topic"
	suffix := "/events/pointset"
	um.config.TopicSuffix = &suffix
	um.config.UDMIVersion = "1.6.0"

	msg, err := um.pointsToPointSet(udmi.PointsEvent{"ok": {PresentValue: 1.0}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var event udmi.PointsetEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if event.Version != "1.6.0" {
		t.Errorf("expected version %q, got %q", "1.6.0", event.Version)
	}
}

func TestStateMessage(t *testing.T) {
	um := &udmiMerge{}
	um.config.TopicPrefix = "client/site-01/HVAC/PICV-12345"
	um.config.Name = "floors/09/devices/FCP_09_01/picv-1"
	um.operational.Store(true)

	msg, err := um.stateMessage()
	if err != nil {
		t.Fatalf("stateMessage: %v", err)
	}
	if msg.Topic != "client/site-01/HVAC/PICV-12345/state" {
		t.Errorf("state topic = %q", msg.Topic)
	}
	var ev udmi.StateEvent
	if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Version != udmi.PointsetVersion || ev.Timestamp.IsZero() {
		t.Errorf("missing version/timestamp: %+v", ev)
	}
	if ev.System.SerialNo != "" {
		t.Errorf("serial_no = %q, want empty (merge devices have no hardware serial)", ev.System.SerialNo)
	}
	if ev.System.Hardware.Make == "" || ev.System.Hardware.Model == "" {
		t.Errorf("hardware make/model required, got %+v", ev.System.Hardware)
	}
	if len(ev.System.Software) == 0 {
		t.Error("software required")
	}
	if !ev.System.Operation.Operational {
		t.Error("expected operational=true")
	}
}

func TestStateMessage_hardwareOverride(t *testing.T) {
	um := &udmiMerge{}
	um.config.TopicPrefix = "client/site-01/HVAC/PICV-1"
	um.config.Hardware = &UdmiHardware{Make: "Tridium", Model: "Niagara JACE"}
	msg, err := um.stateMessage()
	if err != nil {
		t.Fatalf("stateMessage: %v", err)
	}
	var ev udmi.StateEvent
	if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.System.Hardware.Make != "Tridium" || ev.System.Hardware.Model != "Niagara JACE" {
		t.Errorf("hardware override not applied: %+v", ev.System.Hardware)
	}
}

func TestMetadataMessage(t *testing.T) {
	um := &udmiMerge{}
	um.config.TopicPrefix = "client/site-01/HVAC/PICV-12345"
	um.config.Name = "picv-1"

	msg, err := um.metadataMessage()
	if err != nil {
		t.Fatalf("metadataMessage: %v", err)
	}
	if msg.Topic != "client/site-01/HVAC/PICV-12345/metadata.json" {
		t.Errorf("metadata topic = %q", msg.Topic)
	}
	var ev udmi.MetadataEvent
	if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.System.Location == nil || ev.System.Location.Site != "site-01" {
		t.Errorf("site = %+v", ev.System.Location)
	}
	// system.name is the UDMI device id (the topic's device segment), not the
	// Smart Core device name (config.Name = "picv-1").
	if ev.System.Name != "PICV-12345" {
		t.Errorf("system.name = %q, want %q", ev.System.Name, "PICV-12345")
	}
}
