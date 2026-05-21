package mock

import (
	"context"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/control"
	"github.com/smart-core-os/sc-bos/pkg/proto/mockpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// TestForceJSON_PatchSemantics verifies that ForceFunc derived from forceJSON merges only
// the fields present in the JSON, leaving unspecified fields unchanged.
func TestForceJSON_PatchSemantics(t *testing.T) {
	model := occupancysensorpb.NewModel()

	// Set initial state with multiple fields populated.
	_, err := model.SetOccupancy(&occupancysensorpb.Occupancy{
		State:       occupancysensorpb.Occupancy_OCCUPIED,
		PeopleCount: 5,
		Confidence:  0.9,
	})
	if err != nil {
		t.Fatalf("SetOccupancy: %v", err)
	}

	ctrl := control.New()
	ctrl.Register("dev1", "trait.Occupancy", forceJSON(
		func() *occupancysensorpb.Occupancy { return new(occupancysensorpb.Occupancy) },
		func(v *occupancysensorpb.Occupancy, opts ...resource.WriteOption) error {
			_, err := model.SetOccupancy(v, opts...)
			return err
		},
	))

	// Force only state — people_count and confidence should be preserved.
	if err := callForce(ctrl, "dev1", "trait.Occupancy", `{"state":"UNOCCUPIED"}`); err != nil {
		t.Fatalf("ForceTraitValue: %v", err)
	}

	got, err := model.GetOccupancy()
	if err != nil {
		t.Fatalf("GetOccupancy: %v", err)
	}
	if got.State != occupancysensorpb.Occupancy_UNOCCUPIED {
		t.Errorf("state = %v, want UNOCCUPIED", got.State)
	}
	if got.PeopleCount != 5 {
		t.Errorf("people_count = %v, want 5 (should be preserved)", got.PeopleCount)
	}
	if got.Confidence != 0.9 {
		t.Errorf("confidence = %v, want 0.9 (should be preserved)", got.Confidence)
	}
}

// TestForceJSON_FullReplace verifies that supplying all fields replaces the entire resource.
func TestForceJSON_FullReplace(t *testing.T) {
	model := occupancysensorpb.NewModel()

	_, err := model.SetOccupancy(&occupancysensorpb.Occupancy{
		State:       occupancysensorpb.Occupancy_OCCUPIED,
		PeopleCount: 5,
	})
	if err != nil {
		t.Fatalf("SetOccupancy: %v", err)
	}

	ctrl := control.New()
	ctrl.Register("dev1", "trait.Occupancy", forceJSON(
		func() *occupancysensorpb.Occupancy { return new(occupancysensorpb.Occupancy) },
		func(v *occupancysensorpb.Occupancy, opts ...resource.WriteOption) error {
			_, err := model.SetOccupancy(v, opts...)
			return err
		},
	))

	if err := callForce(ctrl, "dev1", "trait.Occupancy", `{"state":"UNOCCUPIED","peopleCount":3}`); err != nil {
		t.Fatalf("ForceTraitValue: %v", err)
	}

	got, err := model.GetOccupancy()
	if err != nil {
		t.Fatalf("GetOccupancy: %v", err)
	}
	if got.State != occupancysensorpb.Occupancy_UNOCCUPIED {
		t.Errorf("state = %v, want UNOCCUPIED", got.State)
	}
	if got.PeopleCount != 3 {
		t.Errorf("people_count = %v, want 3", got.PeopleCount)
	}
}

// TestForceJSON_InvalidJSON verifies that malformed protojson returns an error.
func TestForceJSON_InvalidJSON(t *testing.T) {
	model := occupancysensorpb.NewModel()
	ctrl := control.New()
	ctrl.Register("dev1", "trait.Occupancy", forceJSON(
		func() *occupancysensorpb.Occupancy { return new(occupancysensorpb.Occupancy) },
		func(v *occupancysensorpb.Occupancy, opts ...resource.WriteOption) error {
			_, err := model.SetOccupancy(v, opts...)
			return err
		},
	))

	if err := callForce(ctrl, "dev1", "trait.Occupancy", `{not valid json`); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// callForce invokes ForceTraitValue on the controller for a single device/trait.
func callForce(ctrl *control.Controller, device, trait, json string) error {
	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   device,
		Values: []*mockpb.TraitValue{{Trait: trait, ValueProtojson: json}},
	})
	return err
}
