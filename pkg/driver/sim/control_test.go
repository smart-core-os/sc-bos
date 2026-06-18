package sim

import (
	"context"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/mockpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func forceValue(t *testing.T, c *controller, device string, tn trait.Name, valueJSON string) {
	t.Helper()
	_, err := c.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   device,
		Values: []*mockpb.TraitValue{{Trait: string(tn), ValueProtojson: valueJSON}},
	})
	if err != nil {
		t.Fatalf("ForceTraitValue(%s, %s): %v", device, tn, err)
	}
}

func TestControl_ForcedOccupancyDrivesBuilding(t *testing.T) {
	// 03:00 — the building is empty: occupancy, lights and demand are all at rest.
	start := at(monday, 3, 0)
	b := newTestBuilding(start)
	now := start
	for i := 0; i < 30; i++ { // settle to a night-time baseline
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
	}
	if b.room().Occupants != 0 {
		t.Fatalf("precondition: night Occupants = %d, want 0", b.room().Occupants)
	}

	c := newController(b)
	c.register("pir-01", b.room(), map[trait.Name]roomOverride{trait.OccupancySensor: overrideOccupancy})
	forceValue(t, c, "pir-01", trait.OccupancySensor, `{"peopleCount":50}`)

	// The force is applied on the engine goroutine at the next tick.
	now = now.Add(time.Minute)
	b.Tick(now, time.Minute)
	if got := b.room().Occupants; got != 50 {
		t.Fatalf("after force, Occupants = %d, want 50", got)
	}
	// And the building responds via the coupling: at 03:00 with no daylight and no
	// simulated occupancy, the lights come on and demand climbs above the base load.
	if b.room().LightLevel <= 0 {
		t.Errorf("forced-occupancy LightLevel = %g, want > 0", b.room().LightLevel)
	}
	if b.DemandW <= testBaseLoadW {
		t.Errorf("forced-occupancy DemandW = %g, want above base %d", b.DemandW, testBaseLoadW)
	}

	// Releasing the override hands the room back to the simulation; overnight it empties.
	if _, err := c.SetDeviceAutomation(context.Background(), &mockpb.SetDeviceAutomationsRequest{
		Name:        "pir-01",
		Automations: []*mockpb.TraitAutomation{{Active: true}},
	}); err != nil {
		t.Fatalf("SetDeviceAutomation: %v", err)
	}
	for i := 0; i < 60; i++ {
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
	}
	if got := b.room().Occupants; got != 0 {
		t.Errorf("after release, night Occupants = %d, want 0", got)
	}
}

func TestControl_ForcedSetPointRaisesTemperature(t *testing.T) {
	start := at(monday, 9, 0)
	b := newTestBuilding(start)
	now := start
	b.Tick(now, time.Minute)

	c := newController(b)
	c.register("fcu-01", b.room(), map[trait.Name]roomOverride{trait.AirTemperature: overrideSetPoint})
	t0 := b.room().TempC
	forceValue(t, c, "fcu-01", trait.AirTemperature, `{"temperatureSetPoint":{"valueCelsius":26}}`)

	now = now.Add(time.Minute)
	b.Tick(now, time.Minute)
	if sp := b.room().setPoint(); sp != 26 {
		t.Fatalf("forced setPoint = %g, want 26", sp)
	}
	for i := 0; i < 120; i++ {
		now = now.Add(time.Minute)
		b.Tick(now, time.Minute)
	}
	if b.room().TempC <= t0 {
		t.Errorf("TempC = %g after raising set point to 26 from ~%g, want higher", b.room().TempC, t0)
	}
}

func TestControl_RejectsDerivedAndUnknown(t *testing.T) {
	b := newTestBuilding(at(monday, 9, 0))
	c := newController(b)
	c.register("pir-01", b.room(), map[trait.Name]roomOverride{trait.OccupancySensor: overrideOccupancy})

	// Electric is derived from the simulation; forcing it must be rejected.
	if _, err := c.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   "pir-01",
		Values: []*mockpb.TraitValue{{Trait: string(trait.Electric), ValueProtojson: `{}`}},
	}); err == nil {
		t.Error("forcing a derived trait: want error, got nil")
	}

	// An unregistered device is rejected too.
	if _, err := c.ForceTraitValue(context.Background(), &mockpb.ForceTraitValuesRequest{
		Name:   "no-such-device",
		Values: []*mockpb.TraitValue{{Trait: string(trait.OccupancySensor), ValueProtojson: `{"peopleCount":1}`}},
	}); err == nil {
		t.Error("forcing an unknown device: want error, got nil")
	}
}
