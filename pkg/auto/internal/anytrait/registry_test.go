package anytrait

import (
	"slices"
	"strings"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func TestNames(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("Names() returned no traits")
	}
	if !slices.IsSorted(names) {
		t.Errorf("Names() not sorted: %v", names)
	}
	if !slices.Contains(names, trait.AirTemperature) {
		t.Errorf("Names() missing AirTemperature: %v", names)
	}
	// Every name reported must resolve.
	for _, n := range names {
		if _, err := FindByName(n); err != nil {
			t.Errorf("FindByName(%q) failed for a name from Names(): %v", n, err)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := Validate(trait.AirTemperature); err != nil {
		t.Errorf("Validate(AirTemperature) = %v, want nil", err)
	}

	err := Validate(trait.Name("smartcore.traits.NotATrait"))
	if err == nil {
		t.Fatal("Validate(unsupported) = nil, want error")
	}
	// Error should list the supported set to guide the user.
	if !strings.Contains(err.Error(), string(trait.AirTemperature)) {
		t.Errorf("Validate error does not list supported traits: %v", err)
	}
}
