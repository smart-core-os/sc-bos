package appconf

import (
	"context"
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
