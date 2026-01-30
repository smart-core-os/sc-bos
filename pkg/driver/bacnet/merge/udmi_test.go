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
		"nan":    udmi.PointValue{PresentValue: math.NaN()},
		"posInf": udmi.PointValue{PresentValue: math.Inf(1)},
		"negInf": udmi.PointValue{PresentValue: math.Inf(-1)},
		"ok":     udmi.PointValue{PresentValue: 42.0},
		"str":    udmi.PointValue{PresentValue: "hello"},
	}

	msg, err := um.pointsToPointSet("topic", points)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]udmi.PointValue
	if err := json.Unmarshal([]byte(msg.Payload), &out); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

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
