package sim

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/sim/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/sim/scale"
)

// TestExampleConfig guards the shipped example against config drift: it parses the
// sim driver block from example/config/sim-building and runs the engine briefly.
func TestExampleConfig(t *testing.T) {
	const path = "../../../example/config/sim-building/app.conf.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var app struct {
		Drivers []json.RawMessage `json:"drivers"`
	}
	if err := json.Unmarshal(raw, &app); err != nil {
		t.Fatalf("parse app config: %v", err)
	}

	var cfg config.Root
	found := false
	for _, d := range app.Drivers {
		var probe struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(d, &probe)
		if probe.Type == DriverName {
			if err := json.Unmarshal(d, &cfg); err != nil {
				t.Fatalf("parse sim driver config: %v", err)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no sim driver in example config")
	}

	cfg.Normalise()
	days, err := cfg.WorkingHours.Weekdays()
	if err != nil {
		t.Fatalf("working hours: %v", err)
	}
	scaler := scale.WorkingHours(cfg.WorkingHours.Start, cfg.WorkingHours.End, days)

	b, devices, err := Expand(cfg, scaler, at(monday, 9, 0))
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("no devices generated")
	}

	// Run the engine for a simulated hour and confirm it publishes without panicking
	// and the building draws at least its base load.
	now := at(monday, 9, 0)
	b.Publish(now)
	for i := 0; i < 12; i++ {
		now = now.Add(5 * time.Minute)
		b.Tick(now, 5*time.Minute)
		b.Publish(now)
	}
	if b.DemandW < cfg.BaseLoadW {
		t.Errorf("DemandW = %g, want >= base %g", b.DemandW, cfg.BaseLoadW)
	}
	if b.MeterKWh <= 0 {
		t.Errorf("MeterKWh = %g, want > 0", b.MeterKWh)
	}
}
