package anytrait

import (
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func TestValidate(t *testing.T) {
	if err := Validate(trait.AirTemperature); err != nil {
		t.Errorf("Validate(AirTemperature) = %v, want nil", err)
	}

	if err := Validate(trait.Name("smartcore.traits.NotATrait")); err == nil {
		t.Fatal("Validate(unsupported) = nil, want error")
	}
}
