package app

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/storagepb"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
	gen "github.com/smart-core-os/sc-bos/pkg/proto/storagepb"
)

func TestUpdateSqliteStorageModel(t *testing.T) {
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

	model := storagepb.NewModel()
	updateSqliteStorageModel(ctx, s, model, tmpDir, zap.NewNop())

	got, err := model.GetStorage()
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	if got.Bytes == nil || got.Bytes.Used == nil || *got.Bytes.Used == 0 {
		t.Error("expected non-zero bytes.used")
	}
	if got.Items == nil || got.Items.Used == nil || *got.Items.Used != 3 {
		t.Errorf("expected items.used=3, got %v", got.Items)
	}
}

func TestUpdateSqliteStorageModel_DiskCapacity(t *testing.T) {
	ctx := t.Context()
	tmpDir := t.TempDir()
	s := stores.New(&stores.Config{DataDir: tmpDir})
	t.Cleanup(func() { _ = s.Close() })

	model := storagepb.NewModel()
	updateSqliteStorageModel(ctx, s, model, tmpDir, zap.NewNop())

	got, err := model.GetStorage()
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	if got.Bytes == nil {
		t.Fatal("expected bytes to be populated")
	}
	if got.Bytes.Capacity == nil || *got.Bytes.Capacity == 0 {
		t.Error("expected non-zero bytes.capacity from disk stats")
	}
	if got.Bytes.Utilization == nil {
		t.Error("expected bytes.utilization to be populated")
	}
}

func TestSqliteAdminHandler_Clear(t *testing.T) {
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

	handler := sqliteAdminHandler(ctx, s, 30*24*time.Hour, zap.NewNop())
	resp, err := handler(ctx, gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR)
	if err != nil {
		t.Fatalf("CLEAR: %v", err)
	}
	if resp.FreedItems == nil || resp.FreedItems.Used == nil || *resp.FreedItems.Used != 5 {
		t.Errorf("expected 5 freed items, got %v", resp.FreedItems)
	}

	count, err := db.TotalCount(ctx)
	if err != nil {
		t.Fatalf("TotalCount: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records after clear, got %d", count)
	}
}

func TestSqliteAdminHandler_DeleteOld(t *testing.T) {
	ctx := t.Context()
	retention := 7 * 24 * time.Hour
	s := stores.New(&stores.Config{DataDir: t.TempDir(), HistoryRetention: retention})
	t.Cleanup(func() { _ = s.Close() })

	db, err := s.SqliteHistory(ctx)
	if err != nil {
		t.Fatalf("SqliteHistory: %v", err)
	}

	// Insert a record older than the retention period
	_, err = db.Insert(ctx, sqlitestore.Record{
		Source:     "test",
		Payload:    []byte("old"),
		CreateTime: time.Now().Add(-retention - time.Hour),
	})
	if err != nil {
		t.Fatalf("Insert old record: %v", err)
	}

	// Insert a recent record
	store := db.OpenStore("test")
	if _, err := store.Append(ctx, []byte("new")); err != nil {
		t.Fatalf("Append new record: %v", err)
	}

	handler := sqliteAdminHandler(ctx, s, retention, zap.NewNop())
	resp, err := handler(ctx, gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD)
	if err != nil {
		t.Fatalf("DELETE_OLD: %v", err)
	}
	if resp.FreedItems == nil || resp.FreedItems.Used == nil || *resp.FreedItems.Used != 1 {
		t.Errorf("expected 1 freed item, got %v", resp.FreedItems)
	}

	count, err := db.TotalCount(ctx)
	if err != nil {
		t.Fatalf("TotalCount: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record remaining, got %d", count)
	}
}

func TestHistoryRetention_Default(t *testing.T) {
	got := historyRetention(nil)
	if got != 30*24*time.Hour {
		t.Errorf("expected 30d default, got %v", got)
	}

	got = historyRetention(&stores.Config{})
	if got != 30*24*time.Hour {
		t.Errorf("expected 30d for zero config, got %v", got)
	}

	custom := 14 * 24 * time.Hour
	got = historyRetention(&stores.Config{HistoryRetention: custom})
	if got != custom {
		t.Errorf("expected %v, got %v", custom, got)
	}
}
