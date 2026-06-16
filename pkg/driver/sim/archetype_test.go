package sim

import (
	"slices"
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/driver/sim/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/sim/scale"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

// TestArchetypeTraitsDeclared locks the trait sets declared on the registry. An
// archetype is a named device type composed of one or more SmartCore traits (a
// composite like fcu spans several), and the descriptor's Traits is the single
// source of truth for what it exposes.
func TestArchetypeTraitsDeclared(t *testing.T) {
	cases := map[string][]trait.Name{
		ArchetypeFCU:        {trait.AirTemperature, trait.FanSpeed, trait.OnOff, statuspb.TraitName},
		ArchetypeLighting:   {trait.Light, statuspb.TraitName},
		ArchetypePIR:        {trait.OccupancySensor},
		ArchetypeMeter:      {meterpb.TraitName},
		ArchetypeElectric:   {trait.Electric},
		ArchetypeEnterLeave: {trait.EnterLeaveSensor},
	}
	for typ, want := range cases {
		if got := archetypes[typ].Traits; !slices.Equal(got, want) {
			t.Errorf("%s Traits = %v, want %v", typ, got, want)
		}
	}

	// Every forceable input must be a declared trait (also enforced by a panic in
	// the registry init; asserted here to document the invariant).
	for _, d := range archetypeList {
		declared := make(map[trait.Name]bool, len(d.Traits))
		for _, tn := range d.Traits {
			declared[tn] = true
		}
		for tn := range d.Forceable {
			if !declared[tn] {
				t.Errorf("archetype %q forces trait %q not in its declared Traits %v", d.Type, tn, d.Traits)
			}
		}
	}
}

// TestExpand_AnnouncesDeclaredTraits confirms the declared traits reach a device's
// announced metadata — i.e. the expansion announces HasTrait from the descriptor's
// Traits, not from the Build funcs (which now return trait servers only).
func TestExpand_AnnouncesDeclaredTraits(t *testing.T) {
	cfg := config.Root{
		NamePrefix: "sim/demo",
		Floors: []config.Floor{{
			Name: "g", MaxOccupancy: 10,
			Rooms: []config.Room{{Name: "r", Archetypes: []config.Archetype{{Type: ArchetypeFCU}}}},
		}},
	}
	cfg.Normalise()
	_, devices, err := Expand(cfg, scale.WorkingHours(8, 18, nil), monday)
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}

	n := node.New("test")
	for _, d := range devices {
		n.Announce(d.Name, d.Features...)
	}

	const fcuName = "sim/demo/g/r/fcu-01"
	want := []trait.Name{trait.AirTemperature, trait.FanSpeed, trait.OnOff, statuspb.TraitName}
	for _, dev := range n.ListDevices() {
		if dev.GetName() != fcuName {
			continue
		}
		got := make(map[trait.Name]bool)
		for _, tm := range dev.GetMetadata().GetTraits() {
			got[trait.Name(tm.GetName())] = true
		}
		for _, tn := range want {
			if !got[tn] {
				t.Errorf("%s missing announced trait %q; got %v", fcuName, tn, dev.GetMetadata().GetTraits())
			}
		}
		return
	}
	t.Fatalf("device %q not found in ListDevices", fcuName)
}
