package airtemperaturepb

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
)

func TestMemoryDevice_GetState_Initial(t *testing.T) {
	api := NewMemoryDevice()
	state, err := api.GetAirTemperature(context.Background(), &GetAirTemperatureRequest{Name: "test"})
	if err != nil {
		t.Errorf("error not expected %v", err)
	}
	if diff := cmp.Diff(InitialAirTemperatureState(), state, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected initial value (-want,+got)\n%v", diff)
	}
}

func TestMemoryDevice_UpdateAirTemperature(t *testing.T) {
	api := NewMemoryDevice()
	initialState, _ := api.GetAirTemperature(context.Background(), &GetAirTemperatureRequest{Name: "test"})
	newState := &AirTemperature{
		// fields we can edit
		Mode: AirTemperature_ECO,
		TemperatureGoal: &AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &typespb.Temperature{ValueCelsius: 30},
		},
		// fields we can't edit
		AmbientTemperature: &typespb.Temperature{ValueCelsius: -12},
		AmbientHumidity:    new(float32(12.2)),
	}
	updatedState, err := api.UpdateAirTemperature(context.Background(), &UpdateAirTemperatureRequest{
		Name:  "test",
		State: newState,
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// check the response is what we expect
	// writable fields
	if diff := cmp.Diff(newState.Mode, updatedState.Mode, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() Mode mismatch (-want,+got)\n%v", diff)
	}
	if diff := cmp.Diff(newState.TemperatureGoal, updatedState.TemperatureGoal, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() TemperatureGoal mismatch (-want,+got)\n%v", diff)
	}
	// read-only fields
	if diff := cmp.Diff(initialState.AmbientHumidity, updatedState.AmbientHumidity, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() AmbientHumidity mismatch (-want,+got)\n%v", diff)
	}
	if diff := cmp.Diff(initialState.AmbientTemperature, updatedState.AmbientTemperature, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() AmbientTemperature mismatch (-want,+got)\n%v", diff)
	}
}

func TestMemoryDevice_UpdateAirTemperature_Mask(t *testing.T) {
	api := NewMemoryDevice()
	initialState, _ := api.GetAirTemperature(context.Background(), &GetAirTemperatureRequest{Name: "test"})
	newState := &AirTemperature{
		// fields we can edit
		Mode: AirTemperature_ECO,
		TemperatureGoal: &AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &typespb.Temperature{ValueCelsius: 30},
		},
		// fields we can't edit
		AmbientTemperature: &typespb.Temperature{ValueCelsius: -12},
		AmbientHumidity:    new(float32(12.2)),
	}
	updatedState, err := api.UpdateAirTemperature(context.Background(), &UpdateAirTemperatureRequest{
		Name:       "test",
		State:      newState,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"mode"}},
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// check the response is what we expect
	// writable fields
	if diff := cmp.Diff(newState.Mode, updatedState.Mode, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() Mode mismatch (-want,+got)\n%v", diff)
	}
	// unedited
	if diff := cmp.Diff(initialState.TemperatureGoal, updatedState.TemperatureGoal, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() TemperatureGoal mismatch (-want,+got)\n%v", diff)
	}
	// read-only fields
	if diff := cmp.Diff(initialState.AmbientHumidity, updatedState.AmbientHumidity, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() AmbientHumidity mismatch (-want,+got)\n%v", diff)
	}
	if diff := cmp.Diff(initialState.AmbientTemperature, updatedState.AmbientTemperature, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateAirTemperature() AmbientTemperature mismatch (-want,+got)\n%v", diff)
	}
}
