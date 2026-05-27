package control_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/mock/control"
	"github.com/smart-core-os/sc-bos/pkg/proto/mockpb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// newLifecycle returns a minimal Lifecycle useful for testing start/stop behaviour.
func newLifecycle() service.Lifecycle {
	return service.New[struct{}](nil)
}

// forceFn returns a ForceFunc that appends the JSON it receives to calls.
func recordingForce(calls *[]string) control.ForceFunc {
	return func(json string) error {
		*calls = append(*calls, json)
		return nil
	}
}

// failForce returns a ForceFunc that always returns err.
func failForce(err error) control.ForceFunc {
	return func(string) error { return err }
}

// --- ForceTraitValue ---

func TestForceTraitValue_DeviceNotFound(t *testing.T) {
	ctrl := control.New()
	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   "dev1",
		Values: []*mockpb.TraitValue{{Trait: "trait.A", ValueProtojson: "{}"}},
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("got %v, want NotFound", got)
	}
}

func TestForceTraitValue_TraitNotFound(t *testing.T) {
	ctrl := control.New()
	ctrl.Register("dev1", "trait.A", recordingForce(new([]string)))

	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   "dev1",
		Values: []*mockpb.TraitValue{{Trait: "trait.OTHER", ValueProtojson: "{}"}},
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("got %v, want NotFound", got)
	}
}

func TestForceTraitValue_CallsForceFunc(t *testing.T) {
	ctrl := control.New()
	var calls []string
	ctrl.Register("dev1", "trait.A", recordingForce(&calls))

	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   "dev1",
		Values: []*mockpb.TraitValue{{Trait: "trait.A", ValueProtojson: `{"x":1}`}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0] != `{"x":1}` {
		t.Fatalf("calls = %v, want [{\"x\":1}]", calls)
	}
}

func TestForceTraitValue_FuncErrorBecomesInvalidArgument(t *testing.T) {
	ctrl := control.New()
	ctrl.Register("dev1", "trait.A", failForce(errors.New("bad value")))

	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   "dev1",
		Values: []*mockpb.TraitValue{{Trait: "trait.A", ValueProtojson: "{}"}},
	})
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", got)
	}
}

func TestForceTraitValue_MultipleValues_AllApplied(t *testing.T) {
	ctrl := control.New()
	var calls []string
	ctrl.Register("dev1", "trait.A", recordingForce(&calls))
	ctrl.Register("dev1", "trait.B", recordingForce(&calls))

	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name: "dev1",
		Values: []*mockpb.TraitValue{
			{Trait: "trait.A", ValueProtojson: `"a"`},
			{Trait: "trait.B", ValueProtojson: `"b"`},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %v", calls)
	}
}

func TestForceTraitValue_TraitNotFound_NothingApplied(t *testing.T) {
	// If a trait in the request is not found, no ForceFunc should be called.
	ctrl := control.New()
	var calls []string
	ctrl.Register("dev1", "trait.A", recordingForce(&calls))
	// trait.MISSING is not registered

	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name: "dev1",
		Values: []*mockpb.TraitValue{
			{Trait: "trait.A", ValueProtojson: `"a"`},
			{Trait: "trait.MISSING", ValueProtojson: `"b"`},
		},
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("got %v, want NotFound", got)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no calls before error, got %v", calls)
	}
}

func TestForceTraitValue_FailFast_SubsequentFuncsNotCalled(t *testing.T) {
	// If the first ForceFunc errors, the second should not be called.
	ctrl := control.New()
	var calls []string
	ctrl.Register("dev1", "trait.A", failForce(errors.New("fail")))
	ctrl.Register("dev1", "trait.B", recordingForce(&calls))

	ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{ //nolint:errcheck
		Name: "dev1",
		Values: []*mockpb.TraitValue{
			{Trait: "trait.A", ValueProtojson: "{}"},
			{Trait: "trait.B", ValueProtojson: "{}"},
		},
	})
	if len(calls) != 0 {
		t.Fatalf("trait.B should not be called after trait.A fails, got calls %v", calls)
	}
}

func TestRegister_UndoRemovesForceFunc(t *testing.T) {
	ctrl := control.New()
	undo := ctrl.Register("dev1", "trait.A", recordingForce(new([]string)))
	undo()

	_, err := ctrl.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   "dev1",
		Values: []*mockpb.TraitValue{{Trait: "trait.A", ValueProtojson: "{}"}},
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("after undo got %v, want NotFound", got)
	}
}

// --- SetDeviceAutomation ---

func TestSetDeviceAutomation_EmptyList_NoOp(t *testing.T) {
	ctrl := control.New()
	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name:        "dev1",
		Automations: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDeviceAutomation_TraitNotFound(t *testing.T) {
	ctrl := control.New()
	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.MISSING", Active: true},
		},
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("got %v, want NotFound", got)
	}
}

func TestSetDeviceAutomation_EmptyTrait_NoAutomations_NoError(t *testing.T) {
	// Empty trait means "all traits"; if nothing is registered it should not error.
	ctrl := control.New()
	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "", Active: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDeviceAutomation_StartsAutomation(t *testing.T) {
	ctrl := control.New()
	slc := newLifecycle()
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slc)

	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.A", Id: "auto1", Active: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slc.State().Active {
		t.Fatal("lifecycle should be active after Start")
	}
}

func TestSetDeviceAutomation_StopsAutomation(t *testing.T) {
	ctrl := control.New()
	slc := newLifecycle()
	slc.Start() //nolint:errcheck
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slc)

	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.A", Id: "auto1", Active: false},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slc.State().Active {
		t.Fatal("lifecycle should be inactive after Stop")
	}
}

func TestSetDeviceAutomation_AlreadyStarted_NoError(t *testing.T) {
	ctrl := control.New()
	slc := newLifecycle()
	slc.Start() //nolint:errcheck
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slc)

	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.A", Id: "auto1", Active: true},
		},
	})
	if err != nil {
		t.Fatalf("ErrAlreadyStarted should not surface as an RPC error, got: %v", err)
	}
}

func TestSetDeviceAutomation_AlreadyStopped_NoError(t *testing.T) {
	ctrl := control.New()
	slc := newLifecycle() // starts inactive
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slc)

	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.A", Id: "auto1", Active: false},
		},
	})
	if err != nil {
		t.Fatalf("ErrAlreadyStopped should not surface as an RPC error, got: %v", err)
	}
}

func TestSetDeviceAutomation_EmptyTrait_MatchesAll(t *testing.T) {
	ctrl := control.New()
	slcA := newLifecycle()
	slcB := newLifecycle()
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slcA)
	ctrl.RegisterLifecycle("dev1", "trait.B", "auto1", slcB)

	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "", Active: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slcA.State().Active || !slcB.State().Active {
		t.Fatalf("both automations should be active: A=%v B=%v", slcA.State().Active, slcB.State().Active)
	}
}

func TestSetDeviceAutomation_EmptyId_MatchesAllForTrait(t *testing.T) {
	ctrl := control.New()
	slc1 := newLifecycle()
	slc2 := newLifecycle()
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slc1)
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto2", slc2)

	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.A", Id: "", Active: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slc1.State().Active || !slc2.State().Active {
		t.Fatalf("both automations for trait.A should be active: auto1=%v auto2=%v", slc1.State().Active, slc2.State().Active)
	}
}

func TestSetDeviceAutomation_SpecificTrait_DoesNotAffectOthers(t *testing.T) {
	ctrl := control.New()
	slcA := newLifecycle()
	slcB := newLifecycle()
	ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slcA)
	ctrl.RegisterLifecycle("dev1", "trait.B", "auto1", slcB)

	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.A", Active: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slcA.State().Active {
		t.Fatal("trait.A should be active")
	}
	if slcB.State().Active {
		t.Fatal("trait.B should not be affected")
	}
}

func TestRegisterLifecycle_UndoRemoves(t *testing.T) {
	ctrl := control.New()
	slc := newLifecycle()
	undo := ctrl.RegisterLifecycle("dev1", "trait.A", "auto1", slc)
	undo()

	// After undo, specifying the trait should give NotFound since nothing is registered.
	_, err := ctrl.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name: "dev1",
		Automations: []*mockpb.TraitAutomation{
			{Trait: "trait.A", Active: true},
		},
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("after undo got %v, want NotFound", got)
	}
}
