package task

import (
	"context"
	"errors"
	"testing"
)

// noopTask runs once and stops without error.
func noopTask(context.Context) (Next, error) {
	return StopNow, nil
}

func TestGroup_Spawn(t *testing.T) {
	var g Group

	gt, err := g.Spawn(context.Background(), "a", noopTask)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gt == nil {
		t.Fatal("expected a GroupTask, got nil")
	}
	if gt.Tag() != "a" {
		t.Errorf("unexpected tag %q", gt.Tag())
	}
	if _, ok := g.Tasks()["a"]; !ok {
		t.Error("spawned task not present in group")
	}
}

func TestGroup_Spawn_duplicateTag(t *testing.T) {
	var g Group

	first, err := g.Spawn(context.Background(), "a", noopTask)
	if err != nil {
		t.Fatalf("unexpected error on first spawn: %v", err)
	}

	dup, err := g.Spawn(context.Background(), "a", noopTask)
	if !errors.Is(err, ErrTagExists) {
		t.Errorf("expected ErrTagExists, got %v", err)
	}
	if dup != nil {
		t.Error("expected nil GroupTask on duplicate tag")
	}

	// The original task must be untouched and still the only one tracked.
	tasks := g.Tasks()
	if len(tasks) != 1 {
		t.Errorf("expected 1 tracked task, got %d", len(tasks))
	}
	if tasks["a"] != first {
		t.Error("duplicate spawn replaced the original task")
	}
}
