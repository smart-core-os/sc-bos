package dataretention

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
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
	updateSqliteModel(ctx, s, model, tmpDir, zap.NewNop())

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
}

func TestUpdateSqliteModel_DiskCapacity(t *testing.T) {
	ctx := t.Context()
	tmpDir := t.TempDir()
	s := stores.New(&stores.Config{DataDir: tmpDir})
	t.Cleanup(func() { _ = s.Close() })

	model := dataretentionpb.NewModel()
	updateSqliteModel(ctx, s, model, tmpDir, zap.NewNop())

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
