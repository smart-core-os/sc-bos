package appconf

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

func TestDriverStore_SaveConfig(t *testing.T) {
	store := setupStoreServiceTest(t)
	driverStore := store.Drivers()
	err := driverStore.SaveConfig(context.TODO(), "foodriver", "",
		[]byte(`{"name": "foodriver", "type": "bar", "property": "bazmodified"}`))
	if err != nil {
		t.Fatal(err)
	}

	expect := Config{
		Drivers: []driver.RawConfig{
			{
				BaseConfig: driver.BaseConfig{Name: "foodriver", Type: "bar"},
				Raw:        []byte(`{"name": "foodriver", "type": "bar", "property": "bazmodified"}`),
			},
		},
		Automation: []auto.RawConfig{
			{
				Config: auto.Config{Name: "fooauto", Type: "bar"},
				Raw:    []byte(`{"name": "fooauto", "type": "bar", "property": "baz"}`),
			},
		},
		Zones: []zone.RawConfig{
			{
				Config: zone.Config{Name: "foozone", Type: "bar"},
				Raw:    []byte(`{"name": "foozone", "type": "bar", "property": "baz"}`),
			},
		},
	}

	active := store.Active()
	if diff := cmp.Diff(expect, active); diff != "" {
		t.Fatalf("unexpected active config (-want +got)\n: %s", diff)
	}
}

func TestAutomationStore_SaveConfig(t *testing.T) {
	store := setupStoreServiceTest(t)
	autoStore := store.Automations()

	err := autoStore.SaveConfig(context.TODO(), "fooauto", "",
		[]byte(`{"name": "fooauto", "type": "bar", "property": "bazmodified"}`))
	if err != nil {
		t.Fatal(err)
	}

	expect := Config{
		Drivers: []driver.RawConfig{
			{
				BaseConfig: driver.BaseConfig{Name: "foodriver", Type: "bar"},
				Raw:        []byte(`{"name": "foodriver", "type": "bar", "property": "baz"}`),
			},
		},
		Automation: []auto.RawConfig{
			{
				Config: auto.Config{Name: "fooauto", Type: "bar"},
				Raw:    []byte(`{"name": "fooauto", "type": "bar", "property": "bazmodified"}`),
			},
		},
		Zones: []zone.RawConfig{
			{
				Config: zone.Config{Name: "foozone", Type: "bar"},
				Raw:    []byte(`{"name": "foozone", "type": "bar", "property": "baz"}`),
			},
		},
	}

	if diff := cmp.Diff(expect, store.Active()); diff != "" {
		t.Fatalf("unexpected active config (-want +got)\n: %s", diff)
	}
}

func TestZoneStore_SaveConfig(t *testing.T) {
	store := setupStoreServiceTest(t)
	zoneStore := store.Zones()
	err := zoneStore.SaveConfig(context.TODO(), "foozone", "",
		[]byte(`{"name": "foozone", "type": "bar", "property": "bazmodified"}`))
	if err != nil {
		t.Fatal(err)
	}

	expect := Config{
		Drivers: []driver.RawConfig{
			{
				BaseConfig: driver.BaseConfig{Name: "foodriver", Type: "bar"},
				Raw:        []byte(`{"name": "foodriver", "type": "bar", "property": "baz"}`),
			},
		},
		Automation: []auto.RawConfig{
			{
				Config: auto.Config{Name: "fooauto", Type: "bar"},
				Raw:    []byte(`{"name": "fooauto", "type": "bar", "property": "baz"}`),
			},
		},
		Zones: []zone.RawConfig{
			{
				Config: zone.Config{Name: "foozone", Type: "bar"},
				Raw:    []byte(`{"name": "foozone", "type": "bar", "property": "bazmodified"}`),
			},
		},
	}

	if diff := cmp.Diff(expect, store.Active()); diff != "" {
		t.Fatalf("unexpected active config (-want +got)\n: %s", diff)
	}
}

func setupStoreServiceTest(t *testing.T) *Store {
	t.Helper()
	external := Config{
		Drivers: []driver.RawConfig{
			{
				BaseConfig: driver.BaseConfig{Name: "foodriver", Type: "bar"},
				Raw:        []byte(`{"name": "foodriver", "type": "bar", "property": "baz"}`),
			},
		},
		Automation: []auto.RawConfig{
			{
				Config: auto.Config{Name: "fooauto", Type: "bar"},
				Raw:    []byte(`{"name": "fooauto", "type": "bar", "property": "baz"}`),
			},
		},
		Zones: []zone.RawConfig{
			{
				Config: zone.Config{Name: "foozone", Type: "bar"},
				Raw:    []byte(`{"name": "foozone", "type": "bar", "property": "baz"}`),
			},
		},
	}
	dir := t.TempDir()
	store, err := LoadStore(external, Schema{}, dir, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

// The second LoadStore on a directory merges through block.Diff and block.ApplyPatches,
// which used to round-trip numbers through float64 and corrupt integers above 2^53 (SCB-1427).
func TestLoadStore_KeepsLargeIntegers(t *testing.T) {
	const bigID = "1308675217947965579"
	external := func(property string) Config {
		return Config{
			Drivers: []driver.RawConfig{
				{
					BaseConfig: driver.BaseConfig{Name: "foodriver", Type: "bar"},
					Raw:        []byte(`{"name":"foodriver","type":"bar","deviceId":` + bigID + `,"property":"` + property + `"}`),
				},
			},
		}
	}
	dir := t.TempDir()
	if _, err := LoadStore(external("first"), Schema{}, dir, zap.NewNop()); err != nil {
		t.Fatalf("first LoadStore: %v", err)
	}
	store, err := LoadStore(external("second"), Schema{}, dir, zap.NewNop())
	if err != nil {
		t.Fatalf("second LoadStore: %v", err)
	}

	drivers := store.Active().Drivers
	if len(drivers) != 1 {
		t.Fatalf("got %d drivers, want 1", len(drivers))
	}
	raw := string(drivers[0].Raw)
	if !strings.Contains(raw, `"deviceId":`+bigID) {
		t.Errorf("active driver config lost the exact deviceId %s: %s", bigID, raw)
	}
	if !strings.Contains(raw, `"property":"second"`) {
		t.Errorf("active driver config did not take the external change: %s", raw)
	}
}
