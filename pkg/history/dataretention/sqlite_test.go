package dataretention

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func TestUpdateSqliteModel(t *testing.T) {
	ctx := t.Context()
	tmpDir := t.TempDir()
	s := stores.New(&stores.Config{DataDir: tmpDir})
	t.Cleanup(func() { _ = s.Close() })

	db, err := s.SqliteHistory(ctx)
	if err != nil {
		t.Fatalf("SqliteHistory: %v", err)
	}
	store := db.OpenStore("test")
	for i := 0; i < 3; i++ {
		if _, err := store.Append(ctx, []byte("payload")); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	model := dataretentionpb.NewModel()
	updateSqliteModel(ctx, s, model, tmpDir, nil, true, zap.NewNop())

	got, err := model.GetDataRetention()
	if err != nil {
		t.Fatalf("GetDataRetention: %v", err)
	}
	if got.Bytes == nil || got.Bytes.Used == nil || *got.Bytes.Used == 0 {
		t.Error("expected non-zero bytes.used")
	}
	if got.Items == nil || got.Items.Used == nil || *got.Items.Used != 3 {
		t.Errorf("expected items.used=3, got %v", got.Items)
	}

	// A non-full update must not re-count, but must keep the last known count.
	if _, err := store.Append(ctx, []byte("payload")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	updateSqliteModel(ctx, s, model, tmpDir, nil, false, zap.NewNop())
	got, err = model.GetDataRetention()
	if err != nil {
		t.Fatalf("GetDataRetention: %v", err)
	}
	if got.Items == nil || got.Items.Used == nil || *got.Items.Used != 3 {
		t.Errorf("expected items.used to stay at the last counted value 3, got %v", got.Items)
	}
	updateSqliteModel(ctx, s, model, tmpDir, nil, true, zap.NewNop())
	got, err = model.GetDataRetention()
	if err != nil {
		t.Fatalf("GetDataRetention: %v", err)
	}
	if got.Items == nil || got.Items.Used == nil || *got.Items.Used != 4 {
		t.Errorf("expected items.used=4 after a full update, got %v", got.Items)
	}
}

func TestUpdateSqliteModel_DiskCapacity(t *testing.T) {
	ctx := t.Context()
	tmpDir := t.TempDir()
	s := stores.New(&stores.Config{DataDir: tmpDir})
	t.Cleanup(func() { _ = s.Close() })

	model := dataretentionpb.NewModel()
	updateSqliteModel(ctx, s, model, tmpDir, nil, true, zap.NewNop())

	got, err := model.GetDataRetention()
	if err != nil {
		t.Fatalf("GetDataRetention: %v", err)
	}
	if got.Bytes == nil {
		t.Fatal("expected bytes to be populated")
	}
	if got.Bytes.Capacity == nil || *got.Bytes.Capacity == 0 {
		t.Error("expected non-zero bytes.capacity from disk stats")
	}
	if got.Bytes.OtherUsed == nil {
		t.Error("expected bytes.other_used to be populated from disk stats")
	}
	if got.Bytes.Available == nil {
		t.Error("expected bytes.available to be populated from disk stats")
	}
}

func TestStorageHealth_Update(t *testing.T) {
	ctx := t.Context()

	// capture the latest health check state as the registry observes updates
	var latest *healthpb.HealthCheck
	registry := healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(_ string, c *healthpb.HealthCheck) *healthpb.HealthCheck {
			latest = c
			return c
		}),
		healthpb.WithOnCheckUpdate(func(_ string, c *healthpb.HealthCheck) {
			latest = c
		}),
	)
	checks := registry.ForOwner("stores")

	h := &storageHealth{checks: checks, name: "test/stores/history", highPct: 90, logger: zap.NewNop()}

	// First update creates the check; utilisation below the threshold -> NORMAL.
	h.update(ctx, 42.5)
	if h.check == nil {
		t.Fatal("expected storage health check to be created on first update")
	}
	if got := latest.GetNormality(); got != healthpb.HealthCheck_NORMAL {
		t.Errorf("expected NORMAL at 42.5%% (threshold 90%%), got %v", got)
	}
	if got := latest.GetBounds().GetDisplayUnit(); got != "%" {
		t.Errorf("expected display unit %q, got %q", "%", got)
	}

	// Utilisation above the threshold -> HIGH.
	h.update(ctx, 95)
	if got := latest.GetNormality(); got != healthpb.HealthCheck_HIGH {
		t.Errorf("expected HIGH at 95%% (threshold 90%%), got %v", got)
	}
	if cv := latest.GetBounds().GetCurrentValue().GetFloatValue(); cv != 95 {
		t.Errorf("expected current value 95, got %v", cv)
	}

	// Back below the threshold -> NORMAL again.
	h.update(ctx, 10)
	if got := latest.GetNormality(); got != healthpb.HealthCheck_NORMAL {
		t.Errorf("expected NORMAL at 10%% (threshold 90%%), got %v", got)
	}
}

func TestStorageHealth_Update_NilSafe(t *testing.T) {
	ctx := t.Context()
	// nil checks (no registry) and nil receiver must not panic.
	(&storageHealth{logger: zap.NewNop()}).update(ctx, 50)
	var h *storageHealth
	h.update(ctx, 50)
	h.dispose()
}

func TestSqliteBackend_Purge_All(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	db, err := s.SqliteHistory(ctx)
	if err != nil {
		t.Fatalf("SqliteHistory: %v", err)
	}
	store := db.OpenStore("test")
	for i := 0; i < 5; i++ {
		if _, err := store.Append(ctx, []byte("payload")); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	b := &sqliteBackend{stores: s}
	freed, err := b.Purge(ctx, nil)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if freed != 5 {
		t.Errorf("expected freed=5, got %d", freed)
	}

	count, err := db.TotalCount(ctx)
	if err != nil {
		t.Fatalf("TotalCount: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records after purge, got %d", count)
	}
}

func TestSqliteBackend_Purge_WithBefore(t *testing.T) {
	ctx := t.Context()
	retention := 7 * 24 * time.Hour
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	db, err := s.SqliteHistory(ctx)
	if err != nil {
		t.Fatalf("SqliteHistory: %v", err)
	}

	_, err = db.Insert(ctx, sqlitestore.Record{
		Source:     "test",
		Payload:    []byte("old"),
		CreateTime: time.Now().Add(-retention - time.Hour),
	})
	if err != nil {
		t.Fatalf("Insert old record: %v", err)
	}

	store := db.OpenStore("test")
	if _, err := store.Append(ctx, []byte("new")); err != nil {
		t.Fatalf("Append new record: %v", err)
	}

	cutoff := time.Now().Add(-retention)
	b := &sqliteBackend{stores: s}
	freed, err := b.Purge(ctx, &cutoff)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if freed != 1 {
		t.Errorf("expected freed=1, got %d", freed)
	}

	count, err := db.TotalCount(ctx)
	if err != nil {
		t.Fatalf("TotalCount: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record remaining, got %d", count)
	}
}
