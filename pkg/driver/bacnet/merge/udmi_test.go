package merge

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
)

func samplePoints() udmi.PointsEvent {
	return udmi.PointsEvent{
		"nan":    udmi.PointValue{PresentValue: math.NaN()},
		"posInf": udmi.PointValue{PresentValue: math.Inf(1)},
		"negInf": udmi.PointValue{PresentValue: math.Inf(-1)},
		"ok":     udmi.PointValue{PresentValue: 42.0},
		"str":    udmi.PointValue{PresentValue: "hello"},
	}
}

func assertSanitised(t *testing.T, out udmi.PointsEvent) {
	t.Helper()
	if out["nan"].PresentValue != nil {
		t.Errorf("expected nan to be nil, got %v", out["nan"].PresentValue)
	}
	if out["posInf"].PresentValue != nil {
		t.Errorf("expected posInf to be nil, got %v", out["posInf"].PresentValue)
	}
	if out["negInf"].PresentValue != nil {
		t.Errorf("expected negInf to be nil, got %v", out["negInf"].PresentValue)
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

	msg, err := um.pointsToPointSet(samplePoints())
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
	assertSanitised(t, event.Points)
}

// A configured udmiVersion overrides the default stamped into the envelope.
func Test_pointsToPointSet_versionConfigurable(t *testing.T) {
	um := &udmiMerge{}
	um.config.TopicPrefix = "topic"
	suffix := "/events/pointset"
	um.config.TopicSuffix = &suffix
	um.config.UDMIVersion = "1.6.0"

	msg, err := um.pointsToPointSet(samplePoints())
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

// The legacy default topic keeps the bare points-map shape for backward compatibility.
func Test_pointsToPointSet_legacyTopicStaysBare(t *testing.T) {
	um := &udmiMerge{}
	um.config.TopicPrefix = "topic"

	msg, err := um.pointsToPointSet(samplePoints())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out udmi.PointsEvent
	if err := json.Unmarshal([]byte(msg.Payload), &out); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	assertSanitised(t, out)
}
