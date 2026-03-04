package service

import (
	"context"
	"errors"
	"testing"
)

var errUnsupportedKind = errors.New("unsupported kind")

func factoryWithError(id, kind string) (Lifecycle, error) {
	return nil, errUnsupportedKind
}

func factoryOK(id, kind string) (Lifecycle, error) {
	return New[struct{}](func(_ context.Context, _ struct{}) error { return nil }), nil
}

// TestMap_Create_FactoryError verifies that when the factory returns an error the service is still
// added to the map with the error visible via State, an ADD change event is emitted, and the
// caller still receives the factory error.
func TestMap_Create_FactoryError(t *testing.T) {
	m := NewMap(factoryWithError, IdIsKind)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	records, changes := m.GetAndListen(ctx)
	if len(records) != 0 {
		t.Fatalf("expected empty map, got %d records", len(records))
	}

	id, state, err := m.Create("svc1", "badkind", State{Active: true})
	if !errors.Is(err, errUnsupportedKind) {
		t.Fatalf("expected errUnsupportedKind, got %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id")
	}
	if state.Err == nil {
		t.Fatal("expected state.Err to be set")
	}
	if state.Active {
		t.Fatal("expected state.Active to be false")
	}

	// Service must be in the map.
	r := m.Get("svc1")
	if r == nil {
		t.Fatal("expected service to be in the map after factory error")
	}
	if r.Id != "svc1" {
		t.Errorf("expected id svc1, got %s", r.Id)
	}
	if r.Service.State().Err == nil {
		t.Error("expected service state to carry the factory error")
	}

	// An ADD change event must have been emitted.
	change := <-changes
	if change.NewValue == nil || change.NewValue.Id != "svc1" {
		t.Errorf("expected ADD change for svc1, got %+v", change)
	}
}

// TestMap_Create_FactoryError_ImmutableMap verifies that an immutable map (nil create func) still
// returns r==nil and does not panic.
func TestMap_Create_FactoryError_ImmutableMap(t *testing.T) {
	m := NewMapOf(nil) // no create func → ErrImmutable

	id, _, err := m.Create("x", "anykind", State{})
	if !errors.Is(err, ErrImmutable) {
		t.Fatalf("expected ErrImmutable, got %v", err)
	}
	if id != "" {
		t.Errorf("expected empty id, got %s", id)
	}
}

// TestMap_Create_OK_FactoryError_DoesNotRemove verifies that a successful service creation is not
// affected when a subsequent creation with a bad kind fails.
func TestMap_Create_OK_FactoryError_DoesNotRemove(t *testing.T) {
	var calls int
	m := NewMap(func(id, kind string) (Lifecycle, error) {
		calls++
		if kind == "bad" {
			return nil, errUnsupportedKind
		}
		return factoryOK(id, kind)
	}, IdIsKind)

	_, _, err := m.Create("good", "good", State{})
	if err != nil {
		t.Fatalf("unexpected error creating good service: %v", err)
	}

	_, _, err = m.Create("bad", "bad", State{})
	if !errors.Is(err, errUnsupportedKind) {
		t.Fatalf("expected errUnsupportedKind, got %v", err)
	}

	// Both should be in the map.
	if m.Get("good") == nil {
		t.Error("good service should still be in map")
	}
	if m.Get("bad") == nil {
		t.Error("bad service should be in map with error state")
	}

	if len(m.Values()) != 2 {
		t.Errorf("expected 2 records, got %d", len(m.Values()))
	}
}
