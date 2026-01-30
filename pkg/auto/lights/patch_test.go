package lights

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/lights/config"
)

func TestBrightnessAutomation_processConfig(t *testing.T) {
	ba := &BrightnessAutomation{logger: zap.NewNop()}

	subs := make(map[string]*testSubscriber)
	changes := make(chan Patcher, 5)
	ctx, stop := context.WithCancel(context.Background())
	go func() {
		for range changes {
		} // drain all
	}()
	t.Cleanup(func() {
		stop()
		close(changes)
		for _, sub := range subs {
			sub.Stop(nil)
		}
	})
	occupancySensorSource := source{
		names: func(cfg config.Root) []string {
			return cfg.OccupancySensors
		},
		new: func(name string, logger *zap.Logger) subscriber {
			if old, ok := subs[name]; ok {
				return old
			}
			sub := &testSubscriber{}
			subs[name] = sub
			return sub
		},
	}
	brightnessSensorSource := source{
		names: func(cfg config.Root) []string {
			return cfg.BrightnessSensors
		},
		new: func(name string, logger *zap.Logger) subscriber {
			if old, ok := subs[name]; ok {
				return old
			}
			sub := &testSubscriber{}
			subs[name] = sub
			return sub
		},
	}
	sources := []*source{&occupancySensorSource, &brightnessSensorSource}

	t.Run("add all", func(t *testing.T) {
		totalSources := ba.processConfig(ctx, config.Root{
			BrightnessSensors: []string{"bs1", "bs2"},
			OccupancySensors:  []string{"os1", "os2"},
		}, sources, changes)
		if totalSources != 4 {
			t.Fatalf("totalSources want 4, got %d", totalSources)
		}
		if diff := keyDiff(brightnessSensorSource.runningSources, "bs1", "bs2"); diff != "" {
			t.Fatalf("BrightnessSensors running (-want,+got)\n%s", diff)
		}
		if diff := keyDiff(occupancySensorSource.runningSources, "os1", "os2"); diff != "" {
			t.Fatalf("OccupancySensors running (-want,+got)\n%s", diff)
		}
	})

	t.Run("remove one", func(t *testing.T) {
		totalSources := ba.processConfig(ctx, config.Root{
			BrightnessSensors: []string{"bs2"},
			OccupancySensors:  []string{"os1"},
		}, sources, changes)
		if totalSources != 2 {
			t.Fatalf("totalSources want 2, got %d", totalSources)
		}
		if diff := keyDiff(brightnessSensorSource.runningSources, "bs2"); diff != "" {
			t.Fatalf("BrightnessSensors running (-want,+got)\n%s", diff)
		}
		if diff := keyDiff(occupancySensorSource.runningSources, "os1"); diff != "" {
			t.Fatalf("OccupancySensors running (-want,+got)\n%s", diff)
		}
	})

	t.Run("replace one", func(t *testing.T) {
		totalSources := ba.processConfig(ctx, config.Root{
			BrightnessSensors: []string{"bs3"},
			OccupancySensors:  []string{"os3"},
		}, sources, changes)
		if totalSources != 2 {
			t.Fatalf("totalSources want 2, got %d", totalSources)
		}
		if diff := keyDiff(brightnessSensorSource.runningSources, "bs3"); diff != "" {
			t.Fatalf("BrightnessSensors running (-want,+got)\n%s", diff)
		}
		if diff := keyDiff(occupancySensorSource.runningSources, "os3"); diff != "" {
			t.Fatalf("OccupancySensors running (-want,+got)\n%s", diff)
		}
	})

	t.Run("add more", func(t *testing.T) {
		totalSources := ba.processConfig(ctx, config.Root{
			BrightnessSensors: []string{"bs3", "bs4"},
			OccupancySensors:  []string{"os3", "os4"},
		}, sources, changes)
		if totalSources != 4 {
			t.Fatalf("totalSources want 4, got %d", totalSources)
		}
		if diff := keyDiff(brightnessSensorSource.runningSources, "bs3", "bs4"); diff != "" {
			t.Fatalf("BrightnessSensors running (-want,+got)\n%s", diff)
		}
		if diff := keyDiff(occupancySensorSource.runningSources, "os3", "os4"); diff != "" {
			t.Fatalf("OccupancySensors running (-want,+got)\n%s", diff)
		}
	})
}

func keyDiff[V any](m map[string]V, want ...string) string {
	gotKeys := make([]string, 0, len(m))
	for k := range m {
		gotKeys = append(gotKeys, k)
	}
	sort.Slice(gotKeys, func(i, j int) bool {
		return gotKeys[i] < gotKeys[j]
	})
	return cmp.Diff(want, gotKeys)
}

type testSubscriber struct {
	m    sync.Mutex
	stop func()
	err  error
}

func (t *testSubscriber) Subscribe(ctx context.Context, _ chan<- Patcher) error {
	if t.stop != nil {
		return errors.New("already subscribed")
	}

	t.m.Lock()
	ctx, t.stop = context.WithCancel(ctx)
	t.m.Unlock()

	<-ctx.Done()
	t.m.Lock()
	err := t.err
	t.m.Unlock()
	if err != nil {
		return err
	}
	return ctx.Err()
}

func (t *testSubscriber) Stop(err error) {
	t.m.Lock()
	t.err = err
	if t.stop != nil {
		t.stop()
	}
	t.m.Unlock()
}
