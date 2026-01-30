package bms

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/bms/config"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func TestProcessReadState(t *testing.T) {
	t.Run("occupancy", func(t *testing.T) {
		t.Run("no sensors, no change", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			effects, ttl := lt.run()
			if ttl != 0 {
				t.Errorf("expected ttl 0, got %v", ttl)
			}
			effects.AssertNoUpdates()
		})
		t.Run("no readings, no change", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			lt.cfg.OccupancySensors = []string{"sensor1", "sensor2"}
			effects, ttl := lt.run()
			if ttl != 0 {
				t.Errorf("expected ttl 0, got %v", ttl)
			}
			effects.AssertNoUpdates()
		})
		t.Run("one occupied, updates mode", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			lt.cfg.OccupancySensors = []string{"sensor1", "sensor2"}
			lt.setOccupied("sensor1")
			effects, ttl := lt.run()
			if ttl != 0 {
				t.Errorf("expected ttl 0, got %v", ttl)
			}
			effects.AssertOccupancyModeOnCall()
			effects.AssertNoUpdates()
		})
		t.Run("unoccupied recently, no change", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			lt.setUnoccupiedFor(time.Second, "sensor1", "sensor2")
			effects, ttl := lt.run()
			if ttl != config.DefaultUnoccupiedDelay-time.Second {
				t.Errorf("expected ttl %v, got %v", config.DefaultUnoccupiedDelay, ttl)
			}
			effects.AssertNoUpdates()
		})
		t.Run("missing sensors, use what we have", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			lt.cfg.OccupancySensors = []string{"sensor1", "sensor2"}
			lt.setUnoccupiedFor(time.Second, "sensor1")
			effects, ttl := lt.run()
			if ttl != config.DefaultUnoccupiedDelay-time.Second {
				t.Errorf("expected ttl %v, got %v", config.DefaultUnoccupiedDelay, ttl)
			}
			effects.AssertNoUpdates()
		})
		t.Run("unoccupied for a while", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			lt.setUnoccupiedFor(config.DefaultUnoccupiedDelay+time.Second, "sensor1", "sensor2")
			effects, ttl := lt.run()
			if ttl != 0 {
				t.Errorf("expected ttl 0, got %v", ttl)
			}
			effects.AssertOccupancyModeOffCall()
			effects.AssertNoUpdates()
		})
		t.Run("occupied schedule", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			lt.cfg.OccupiedSchedule = []config.Range{
				{Start: *jsontypes.MustParseSchedule("10 0 * * *"), End: *jsontypes.MustParseSchedule("20 0 * * *")},
			}
			// before schedule
			lt.now = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			effects, ttl := lt.run()
			if ttl != 10*time.Minute {
				t.Errorf("expected ttl 10m, got %v", ttl)
			}
			effects.AssertOccupancyModeOffCall()
			// during schedule
			lt.now = time.Date(2025, time.January, 1, 0, 15, 0, 0, time.UTC)
			effects, ttl = lt.run()
			if ttl != 5*time.Minute {
				t.Errorf("expected ttl 5m, got %v", ttl)
			}
			effects.AssertOccupancyModeOnCall()
			// after schedule
			lt.now = time.Date(2025, time.January, 1, 0, 30, 0, 0, time.UTC)
			effects, ttl = lt.run()
			if ttl != 24*time.Hour-20*time.Minute {
				t.Errorf("expected ttl 24h-20m, got %v", ttl)
			}
			effects.AssertOccupancyModeOffCall()
			effects.AssertNoUpdates()
		})
		t.Run("occupied sensor with schedule", func(t *testing.T) {
			lt := newLogicTester(t)
			lt.controlsOccupancy()
			lt.cfg.OccupiedSchedule = []config.Range{
				{Start: *jsontypes.MustParseSchedule("0 1 * * *"), End: *jsontypes.MustParseSchedule("0 4 * * *")},
			}
			// occupied before schedule start
			lt.setOccupied("sensor1")
			lt.now = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			effects, ttl := lt.run()
			if ttl != time.Hour {
				t.Errorf("expected ttl 1h, got %v", ttl)
			}
			effects.AssertOccupancyModeOffCall()
			// occupied during schedule
			lt.now = time.Date(2025, time.January, 1, 2, 0, 0, 0, time.UTC)
			effects, ttl = lt.run()
			if ttl != 2*time.Hour {
				t.Errorf("expected ttl 2h, got %v", ttl)
			}
			effects.AssertOccupancyModeOnCall()
			// becomes unoccupied within schedule (not long enough yet)
			lt.now = time.Date(2025, time.January, 1, 3, 0, 0, 0, time.UTC)
			lt.setUnoccupiedFor(1*time.Minute, "sensor1")
			effects, ttl = lt.run()
			if ttl != config.DefaultUnoccupiedDelay-time.Minute {
				t.Errorf("expected ttl %v, got %v", config.DefaultUnoccupiedDelay-time.Minute, ttl)
			}
			effects.AssertNoUpdates()
			// is unoccupied for long enough within the schedule
			lt.now = time.Date(2025, time.January, 1, 3, 15, 0, 0, time.UTC)
			lt.setUnoccupiedFor(config.DefaultUnoccupiedDelay+time.Minute, "sensor1")
			effects, ttl = lt.run()
			if ttl != 45*time.Minute {
				t.Errorf("expected ttl 45m, got %v", ttl)
			}
			effects.AssertOccupancyModeOffCall()
			// is occupied after the schedule
			lt.now = time.Date(2025, time.January, 1, 5, 0, 0, 0, time.UTC)
			lt.setOccupied("sensor1")
			effects, ttl = lt.run()
			if ttl != 20*time.Hour {
				t.Errorf("expected ttl 23h, got %v", ttl)
			}
			effects.AssertOccupancyModeOffCall()
			effects.AssertNoUpdates()
		})
	})
}

type logicTester struct {
	t   *testing.T
	now time.Time
	cfg *config.Root
	rs  *ReadState
	ws  *WriteState
}

func newLogicTester(t *testing.T) *logicTester {
	lt := &logicTester{
		t:   t,
		now: time.Unix(0, 0),
		rs:  NewReadState(),
		ws:  NewWriteState(),
	}
	lt.cfg = &lt.rs.Config
	lt.rs.Now = func() time.Time {
		return lt.now
	}
	return lt
}

func (lt *logicTester) run() (*testActions, time.Duration) {
	actions := &testActions{t: lt.t}
	ttl, err := processReadState(context.Background(), lt.rs, lt.ws, actions)
	if err != nil {
		lt.t.Fatal(err)
	}
	return actions, ttl
}

func (lt *logicTester) controlsOccupancy() {
	lt.cfg.OccupancyModeTargets = []config.SwitchMode{
		{Name: "occupancyMode1"},                               // defaults
		{Name: "occupancyMode2", Key: "k2"},                    // default values
		{Name: "occupancyMode3", Key: "k3", On: "y", Off: "n"}, // custom
	}
}

func (lt *logicTester) setOccupied(names ...string) {
	lt.setOccupiedAt(lt.now, names...)
}

func (lt *logicTester) setUnoccupied(names ...string) {
	lt.setUnoccupiedAt(lt.now, names...)
}

func (lt *logicTester) setOccupiedFor(d time.Duration, names ...string) {
	if d < 0 {
		d = -d
	}
	lt.setOccupiedAt(lt.now.Add(d), names...)
}

func (lt *logicTester) setUnoccupiedFor(d time.Duration, names ...string) {
	if d > 0 {
		d = -d
	}
	lt.setUnoccupiedAt(lt.now.Add(d), names...)
}

func (lt *logicTester) setOccupiedAt(t time.Time, names ...string) {
	if missing := collectMissing(lt.cfg.OccupancySensors, names...); len(missing) > 0 {
		lt.cfg.OccupancySensors = append(lt.cfg.OccupancySensors, missing...)
		slices.Sort(lt.cfg.OccupancySensors)
	}
	for _, name := range names {
		v := &traits.Occupancy{State: traits.Occupancy_OCCUPIED}
		if !t.IsZero() {
			v.StateChangeTime = timestamppb.New(t)
		}
		lt.rs.Occupancy[name] = Value[*traits.Occupancy]{
			V:  v,
			At: lt.now,
		}
	}
}

func (lt *logicTester) setUnoccupiedAt(t time.Time, names ...string) {
	if missing := collectMissing(lt.cfg.OccupancySensors, names...); len(missing) > 0 {
		lt.cfg.OccupancySensors = append(lt.cfg.OccupancySensors, missing...)
		slices.Sort(lt.cfg.OccupancySensors)
	}
	for _, name := range names {
		v := &traits.Occupancy{State: traits.Occupancy_UNOCCUPIED}
		if !t.IsZero() {
			v.StateChangeTime = timestamppb.New(t)
		}
		lt.rs.Occupancy[name] = Value[*traits.Occupancy]{
			V:  v,
			At: lt.now,
		}
	}
}

func collectMissing(slice []string, values ...string) []string {
	var missing []string
	for _, v := range values {
		if _, ok := slices.BinarySearch(slice, v); !ok {
			missing = append(missing, v)
		}
	}
	return missing
}

type testActions struct {
	t                     *testing.T
	airTemperatureUpdates []*traits.UpdateAirTemperatureRequest
	modeValuesUpdates     []*traits.UpdateModeValuesRequest
}

func (a *testActions) UpdateAirTemperature(ctx context.Context, req *traits.UpdateAirTemperatureRequest, ws *WriteState) error {
	a.airTemperatureUpdates = append(a.airTemperatureUpdates, req)
	return nil
}

func (a *testActions) UpdateModeValues(ctx context.Context, req *traits.UpdateModeValuesRequest, ws *WriteState) error {
	a.modeValuesUpdates = append(a.modeValuesUpdates, req)
	return nil
}

func (a *testActions) AssertNoAirTemperatureUpdates() {
	a.t.Helper()
	if len(a.airTemperatureUpdates) != 0 {
		a.t.Errorf("expected no air temperature updates, got %d", len(a.airTemperatureUpdates))
	}
}

func (a *testActions) AssertNoModeUpdates() {
	a.t.Helper()
	if len(a.modeValuesUpdates) != 0 {
		a.t.Errorf("expected no mode values updates, got %d", len(a.modeValuesUpdates))
	}
}

func (a *testActions) AssertOccupancyModeOnCall() {
	a.t.Helper()
	want := []*traits.UpdateModeValuesRequest{
		{Name: "occupancyMode1", ModeValues: &traits.ModeValues{Values: map[string]string{"occupancy": "occupied"}}},
		{Name: "occupancyMode2", ModeValues: &traits.ModeValues{Values: map[string]string{"k2": "occupied"}}},
		{Name: "occupancyMode3", ModeValues: &traits.ModeValues{Values: map[string]string{"k3": "y"}}},
	}

	if len(a.modeValuesUpdates) < len(want) {
		a.t.Fatalf("expected at least %d mode values update, got %d", len(want), len(a.modeValuesUpdates))
	}
	got := a.modeValuesUpdates[:len(want)]
	a.modeValuesUpdates = a.modeValuesUpdates[len(want):]
	if diff := cmp.Diff(want, got, protocmp.Transform(), cmpopts.SortSlices(func(a, b *traits.UpdateModeValuesRequest) int {
		return strings.Compare(a.Name, b.Name)
	})); diff != "" {
		a.t.Errorf("unexpected mode values update (-want +got):\n%s", diff)
	}
}

func (a *testActions) AssertOccupancyModeOffCall() {
	a.t.Helper()
	want := []*traits.UpdateModeValuesRequest{
		{Name: "occupancyMode1", ModeValues: &traits.ModeValues{Values: map[string]string{"occupancy": "unoccupied"}}},
		{Name: "occupancyMode2", ModeValues: &traits.ModeValues{Values: map[string]string{"k2": "unoccupied"}}},
		{Name: "occupancyMode3", ModeValues: &traits.ModeValues{Values: map[string]string{"k3": "n"}}},
	}

	if len(a.modeValuesUpdates) < len(want) {
		a.t.Fatalf("expected at least %d mode values update, got %d", len(want), len(a.modeValuesUpdates))
	}
	got := a.modeValuesUpdates[:len(want)]
	a.modeValuesUpdates = a.modeValuesUpdates[len(want):]
	if diff := cmp.Diff(want, got, protocmp.Transform(), cmpopts.SortSlices(func(a, b *traits.UpdateModeValuesRequest) int {
		return strings.Compare(a.Name, b.Name)
	})); diff != "" {
		a.t.Errorf("unexpected mode values update (-want +got):\n%s", diff)
	}
}

func (a *testActions) AssertNoUpdates() {
	a.t.Helper()
	a.AssertNoModeUpdates()
	a.AssertNoAirTemperatureUpdates()
}
