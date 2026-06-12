package sim

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/sim/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/time/clock"
)

// countingAnnouncer records announced names and how many announcements were undone.
type countingAnnouncer struct {
	mu        sync.Mutex
	announced []string
	undone    int
}

func (c *countingAnnouncer) Announce(name string, _ ...node.Feature) node.Undo {
	c.mu.Lock()
	c.announced = append(c.announced, name)
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		c.undone++
		c.mu.Unlock()
	}
}

func newTestDriver(a node.Announcer) *Driver {
	d := &Driver{
		announcer: a,
		clk:       clock.Real(),
		logger:    zap.NewNop(),
	}
	d.Service = service.New(d.applyConfig, service.WithOnStop[config.Root](func() { d.stop() }))
	return d
}

func TestDriver_AnnounceAndStop(t *testing.T) {
	a := &countingAnnouncer{}
	d := newTestDriver(a)

	cfg := config.Root{
		NamePrefix: "sim/demo",
		Floors: []config.Floor{{
			Name: "ground", MaxOccupancy: 30,
			Rooms: []config.Room{{Name: "open-plan", Archetypes: []config.Archetype{
				{Type: ArchetypeLighting, Count: 2},
				{Type: ArchetypePIR},
			}}},
		}},
		BuildingDevices: []config.Archetype{{Type: ArchetypeElectric}},
	}
	cfg.Normalise()

	if err := d.applyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	a.mu.Lock()
	got := len(a.announced)
	a.mu.Unlock()
	if got != 4 { // 2 lighting + 1 pir + 1 electric
		t.Errorf("announced %d devices, want 4", got)
	}

	d.stop()

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.undone != got {
		t.Errorf("undone %d, want all %d announcements reversed", a.undone, got)
	}
}

func TestDriver_InvalidFaultProbability(t *testing.T) {
	d := newTestDriver(&countingAnnouncer{})
	cfg := config.Root{
		Floors:      []config.Floor{{Name: "g", Rooms: []config.Room{{Name: "r"}}}},
		HealthCheck: &config.HealthCheck{FaultProbability: ptr(1.5)},
	}
	cfg.Normalise()
	if err := d.applyConfig(context.Background(), cfg); err == nil {
		t.Fatal("expected error for faultProbability > 1")
	}
}

func TestConfig_FaultProbabilityBounds(t *testing.T) {
	// 0 is a valid probability (never fault) and must be preserved, not defaulted.
	zero := config.Root{HealthCheck: &config.HealthCheck{FaultProbability: ptr(0.0)}}
	zero.Normalise()
	if zero.HealthCheck.FaultProbability == nil || *zero.HealthCheck.FaultProbability != 0 {
		t.Errorf("faultProbability 0 not preserved: %v", zero.HealthCheck.FaultProbability)
	}
	if err := zero.Validate(); err != nil {
		t.Errorf("faultProbability 0 should be valid: %v", err)
	}
	// Omitted -> default applied.
	def := config.Root{HealthCheck: &config.HealthCheck{}}
	def.Normalise()
	if def.HealthCheck.FaultProbability == nil || *def.HealthCheck.FaultProbability != config.DefaultFaultProbability {
		t.Errorf("omitted faultProbability not defaulted to %g", config.DefaultFaultProbability)
	}
	// Negative -> rejected by Validate.
	neg := config.Root{HealthCheck: &config.HealthCheck{FaultProbability: ptr(-0.1)}}
	neg.Normalise()
	if err := neg.Validate(); err == nil {
		t.Error("negative faultProbability should be rejected")
	}
}

func TestConfig_DuplicateNames(t *testing.T) {
	cases := []struct {
		name    string
		cfg     config.Root
		wantErr bool
	}{
		{"duplicate floors", config.Root{Floors: []config.Floor{{Name: "g"}, {Name: "g"}}}, true},
		{"duplicate rooms", config.Root{Floors: []config.Floor{{Name: "g", Rooms: []config.Room{{Name: "r"}, {Name: "r"}}}}}, true},
		{"unnamed floor", config.Root{Floors: []config.Floor{{}}}, true},
		{"unnamed room", config.Root{Floors: []config.Floor{{Name: "g", Rooms: []config.Room{{}}}}}, true},
		{"same room name on different floors", config.Root{Floors: []config.Floor{
			{Name: "g", Rooms: []config.Room{{Name: "r"}}},
			{Name: "f1", Rooms: []config.Room{{Name: "r"}}},
		}}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Normalise()
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_RootMetadata(t *testing.T) {
	a := &countingAnnouncer{}
	d := newTestDriver(a)
	cfg := config.Root{
		NamePrefix: "sim/demo",
		Metadata:   &metadatapb.Metadata{Appearance: &metadatapb.Metadata_Appearance{Title: "Simulated Building"}},
		Floors: []config.Floor{{Name: "g", MaxOccupancy: 10,
			Rooms: []config.Room{{Name: "r", Archetypes: []config.Archetype{{Type: ArchetypePIR}}}}}},
	}
	cfg.Name = "sim/demo/driver"
	cfg.Normalise()

	if err := d.applyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	a.mu.Lock()
	names := append([]string(nil), a.announced...)
	a.mu.Unlock()
	if !slices.Contains(names, "sim/demo/driver") {
		t.Errorf("driver root device not announced; got %v", names)
	}
	if len(names) != 2 { // pir + driver root
		t.Errorf("announced %d devices, want 2", len(names))
	}

	d.stop()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.undone != len(names) {
		t.Errorf("undone %d, want all %d announcements reversed", a.undone, len(names))
	}
}

func TestDriver_RunEngineTimeMultiplier(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDriver(&countingAnnouncer{})
		b := newTestBuilding(d.clk.Now())
		var published []time.Time
		b.AddUpdater(updaterFunc(func(now time.Time, _ *Building) {
			published = append(published, now)
		}))

		cfg := config.Root{TimeMultiplier: 288}
		cfg.Normalise() // default 5s tick => 24 simulated minutes per tick

		ctx, cancel := context.WithCancel(t.Context())
		go d.runEngine(ctx, b, cfg)
		time.Sleep(11 * time.Second) // two ticks fire, at +5s and +10s
		cancel()
		synctest.Wait()

		// One priming publish plus one per tick, each advancing simulated time by
		// the wall tick scaled by the multiplier.
		want := []time.Duration{0, 24 * time.Minute, 48 * time.Minute}
		if len(published) != len(want) {
			t.Fatalf("publish count = %d, want %d", len(published), len(want))
		}
		for i, w := range want {
			if got := published[i].Sub(published[0]); got != w {
				t.Errorf("publish %d sim offset = %v, want %v", i, got, w)
			}
		}
	})
}
