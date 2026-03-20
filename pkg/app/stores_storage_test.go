package app

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

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

	handler := sqliteAdminHandler(ctx, s, zap.NewNop())
	resp, err := handler(ctx, &gen.PerformStorageAdminRequest{Action: gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR})
	if err != nil {
		t.Fatalf("CLEAR: %v", err)
	}
	if resp.FreedItemCount == nil || *resp.FreedItemCount != 5 {
		t.Errorf("expected FreedItemCount=5, got %v", resp.FreedItemCount)
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
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
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

	handler := sqliteAdminHandler(ctx, s, zap.NewNop())
	resp, err := handler(ctx, &gen.PerformStorageAdminRequest{
		Action:          gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD,
		RetentionPeriod: durationpb.New(retention),
	})
	if err != nil {
		t.Fatalf("DELETE_OLD: %v", err)
	}
	if resp.FreedItemCount == nil || *resp.FreedItemCount != 1 {
		t.Errorf("expected FreedItemCount=1, got %v", resp.FreedItemCount)
	}

	count, err := db.TotalCount(ctx)
	if err != nil {
		t.Fatalf("TotalCount: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record remaining, got %d", count)
	}
}

func TestSqliteAdminHandler_DeleteOld_NoRetention(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	handler := sqliteAdminHandler(ctx, s, zap.NewNop())
	_, err := handler(ctx, &gen.PerformStorageAdminRequest{
		Action: gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD,
	})
	if err == nil {
		t.Fatal("expected error when retention_period not set")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestUpdatePostgresStorageModel_NotConfigured(t *testing.T) {
	ctx := t.Context()
	// Stores with no postgres config — should log and return without updating the model.
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	model := storagepb.NewModel()
	updatePostgresStorageModel(ctx, s, model, &postgresRowCountCache{}, zap.NewNop())

	// Model should be empty (default zero value) since postgres is not configured.
	got, err := model.GetStorage()
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	if got.Bytes != nil || got.Items != nil {
		t.Errorf("expected empty storage model when postgres not configured, got %v", got)
	}
}

func TestPostgresAdminHandler_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	handler := postgresAdminHandler(s, zap.NewNop())
	_, err := handler(ctx, &gen.PerformStorageAdminRequest{Action: gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR})
	if err == nil {
		t.Fatal("expected error when postgres not configured, got nil")
	}
}

func TestRetentionFromRequest(t *testing.T) {
	t.Run("nil retention_period errors", func(t *testing.T) {
		_, err := retentionFromRequest(&gen.PerformStorageAdminRequest{})
		if err == nil {
			t.Fatal("expected error for nil retention_period")
		}
		if st, ok := status.FromError(err); !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("zero duration errors", func(t *testing.T) {
		_, err := retentionFromRequest(&gen.PerformStorageAdminRequest{
			RetentionPeriod: durationpb.New(0),
		})
		if err == nil {
			t.Fatal("expected error for zero retention_period")
		}
	})

	t.Run("positive duration returned", func(t *testing.T) {
		want := 48 * time.Hour
		got, err := retentionFromRequest(&gen.PerformStorageAdminRequest{
			RetentionPeriod: durationpb.New(want),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("expected %v, got %v", want, got)
		}
	})
}
