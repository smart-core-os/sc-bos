package sim

import (
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/driver/sim/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/sim/scale"
)

func TestExpand_DeviceGeneration(t *testing.T) {
	cfg := config.Root{
		NamePrefix: "sim/demo",
		Floors: []config.Floor{{
			Name:         "ground",
			MaxOccupancy: 60,
			Rooms: []config.Room{
				{Name: "open-plan", Archetypes: []config.Archetype{
					{Type: ArchetypeLighting, Count: 3},
					{Type: ArchetypePIR},
					{Type: ArchetypeFCU, Count: 2},
				}},
				{Name: "meeting-1", Archetypes: []config.Archetype{
					{Type: ArchetypeLighting, Count: 2},
					{Type: ArchetypePIR},
				}},
			},
		}},
		BuildingDevices: []config.Archetype{
			{Type: ArchetypeMeter},
			{Type: ArchetypeElectric},
		},
	}
	cfg.Normalise()
	scaler := scale.WorkingHours(8, 18, nil)

	b, devices, err := Expand(cfg, scaler, monday)
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}

	// 3+1+2 + 2+1 + 2 = 11 devices.
	if len(devices) != 11 {
		t.Errorf("device count = %d, want 11", len(devices))
	}

	byName := make(map[string]Device, len(devices))
	for _, d := range devices {
		byName[d.Name] = d
	}
	wantSubsystem := map[string]string{
		"sim/demo/ground/open-plan/lighting-01": "lighting",
		"sim/demo/ground/open-plan/lighting-03": "lighting",
		"sim/demo/ground/open-plan/pir-01":      "sensors",
		"sim/demo/ground/open-plan/fcu-02":      "hvac",
		"sim/demo/ground/meeting-1/lighting-02": "lighting",
		"sim/demo/building/meter-01":            "metering",
		"sim/demo/building/electric-01":         "metering",
	}
	for name, sub := range wantSubsystem {
		d, ok := byName[name]
		if !ok {
			t.Errorf("missing device %q", name)
			continue
		}
		if got := d.Metadata.GetMembership().GetSubsystem(); got != sub {
			t.Errorf("%q subsystem = %q, want %q", name, got, sub)
		}
	}

	// Room max occupancy should be shared evenly across the two rooms (60/2 = 30).
	for _, r := range b.Floors[0].Rooms {
		if r.MaxOccupancy != 30 {
			t.Errorf("room %q MaxOccupancy = %d, want 30", r.Name, r.MaxOccupancy)
		}
	}

	// Updaters: one per device that publishes (lighting+fcu+pir = 9 room devices,
	// meter+electric = 2 building devices) — every device here has at least one.
	if len(b.updaters) == 0 {
		t.Error("no updaters registered")
	}
}

func TestExpand_UnknownArchetype(t *testing.T) {
	cfg := config.Root{
		NamePrefix: "sim/demo",
		Floors: []config.Floor{{
			Name: "ground", MaxOccupancy: 10,
			Rooms: []config.Room{{Name: "r1", Archetypes: []config.Archetype{{Type: "nonsense"}}}},
		}},
	}
	cfg.Normalise()
	if _, _, err := Expand(cfg, scale.WorkingHours(8, 18, nil), monday); err == nil {
		t.Fatal("expected error for unknown archetype, got nil")
	}
}

func TestExpand_BuildingLevelRoomArchetypeErrors(t *testing.T) {
	// Room-scoped archetypes have no room when placed in buildingDevices; this must
	// be a config error rather than a nil-pointer panic on the engine goroutine.
	for _, typ := range []string{ArchetypeLighting, ArchetypeFCU, ArchetypePIR, ArchetypeMotion, ArchetypeBrightness, ArchetypeAirQuality} {
		cfg := config.Root{
			NamePrefix:      "sim/demo",
			Floors:          []config.Floor{{Name: "g", MaxOccupancy: 10, Rooms: []config.Room{{Name: "r", Archetypes: []config.Archetype{{Type: ArchetypePIR}}}}}},
			BuildingDevices: []config.Archetype{{Type: typ}},
		}
		cfg.Normalise()
		if _, _, err := Expand(cfg, scale.WorkingHours(8, 18, nil), monday); err == nil {
			t.Errorf("%s as a building device: expected error, got nil", typ)
		}
	}
}

func TestExpand_SameTypeNumberingContinues(t *testing.T) {
	// Two archetype entries of the same type in one room (e.g. to mix titles or
	// rated powers) must not produce colliding device names.
	cfg := config.Root{
		NamePrefix: "sim/demo",
		Floors: []config.Floor{{
			Name: "g", MaxOccupancy: 10,
			Rooms: []config.Room{{Name: "r", Archetypes: []config.Archetype{
				{Type: ArchetypeLighting, Count: 2, Title: "Downlight"},
				{Type: ArchetypeLighting, Title: "Pendant"},
			}}},
		}},
	}
	cfg.Normalise()
	_, devices, err := Expand(cfg, scale.WorkingHours(8, 18, nil), monday)
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}

	want := []string{
		"sim/demo/g/r/lighting-01",
		"sim/demo/g/r/lighting-02",
		"sim/demo/g/r/lighting-03",
	}
	if len(devices) != len(want) {
		t.Fatalf("device count = %d, want %d", len(devices), len(want))
	}
	seen := make(map[string]bool, len(devices))
	for i, d := range devices {
		if d.Name != want[i] {
			t.Errorf("device %d name = %q, want %q", i, d.Name, want[i])
		}
		if seen[d.Name] {
			t.Errorf("duplicate device name %q", d.Name)
		}
		seen[d.Name] = true
	}
	// The title number tracks the device name suffix, not the archetype entry.
	if got := devices[2].Metadata.GetAppearance().GetTitle(); got != "Pendant 3" {
		t.Errorf("third device title = %q, want %q", got, "Pendant 3")
	}
}

func TestZeroOccupancyRooms(t *testing.T) {
	// No maxOccupancy anywhere: the pir room is reported, the lighting-only
	// corridor (sensible without occupants) is not.
	cfg := config.Root{
		Floors: []config.Floor{{
			Name: "g",
			Rooms: []config.Room{
				{Name: "office", Archetypes: []config.Archetype{{Type: ArchetypePIR}}},
				{Name: "corridor", Archetypes: []config.Archetype{{Type: ArchetypeLighting}}},
			},
		}},
	}
	got := zeroOccupancyRooms(cfg)
	if len(got) != 1 || got[0] != "g/office" {
		t.Errorf("zeroOccupancyRooms = %v, want [g/office]", got)
	}

	// With a floor occupancy to share, nothing is reported.
	cfg.Floors[0].MaxOccupancy = 10
	if got := zeroOccupancyRooms(cfg); len(got) != 0 {
		t.Errorf("zeroOccupancyRooms with occupancy = %v, want none", got)
	}
}
