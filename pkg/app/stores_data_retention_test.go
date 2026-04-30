package app

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
)

func TestUpdateSqliteDataRetentionModel(t *testing.T) {
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
	updateSqliteDataRetentionModel(ctx, s, model, tmpDir, zap.NewNop())

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

func TestUpdateSqliteDataRetentionModel_DiskCapacity(t *testing.T) {
	ctx := t.Context()
	tmpDir := t.TempDir()
	s := stores.New(&stores.Config{DataDir: tmpDir})
	t.Cleanup(func() { _ = s.Close() })

	model := dataretentionpb.NewModel()
	updateSqliteDataRetentionModel(ctx, s, model, tmpDir, zap.NewNop())

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
	if got.Bytes.Utilization == nil {
		t.Error("expected bytes.utilization to be populated")
	}
}

func TestSqliteClearHandler(t *testing.T) {
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

	handler := sqliteClearHandler(s)
	resp, err := handler(ctx, &dataretentionpb.ClearDataRetentionRequest{})
	if err != nil {
		t.Fatalf("ClearDataRetention: %v", err)
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

func TestSqliteDeleteOldHandler(t *testing.T) {
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

	handler := sqliteDeleteOldHandler(s)
	resp, err := handler(ctx, &dataretentionpb.DeleteOldDataRetentionRequest{
		RetentionPeriod: durationpb.New(retention),
	})
	if err != nil {
		t.Fatalf("DeleteOldDataRetention: %v", err)
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

func TestSqliteDeleteOldHandler_NoRetention(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	handler := sqliteDeleteOldHandler(s)
	_, err := handler(ctx, &dataretentionpb.DeleteOldDataRetentionRequest{})
	if err == nil {
		t.Fatal("expected error when retention_period not set")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestUpdatePostgresDataRetentionModel_NotConfigured(t *testing.T) {
	ctx := t.Context()
	// Stores with no postgres config — should log and return without updating the model.
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	model := dataretentionpb.NewModel()
	updatePostgresDataRetentionModel(ctx, s, model, &postgresRowCountCache{}, zap.NewNop())

	// Model should be empty (default zero value) since postgres is not configured.
	got, err := model.GetDataRetention()
	if err != nil {
		t.Fatalf("GetDataRetention: %v", err)
	}
	if got.Bytes != nil || got.Items != nil {
		t.Errorf("expected empty data retention model when postgres not configured, got %v", got)
	}
}

func TestPostgresClearHandler_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	handler := postgresClearHandler(s)
	_, err := handler(ctx, &dataretentionpb.ClearDataRetentionRequest{})
	if err == nil {
		t.Fatal("expected error when postgres not configured, got nil")
	}
}

func TestPostgresDeleteOldHandler_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	handler := postgresDeleteOldHandler(s)
	_, err := handler(ctx, &dataretentionpb.DeleteOldDataRetentionRequest{
		RetentionPeriod: durationpb.New(24 * time.Hour),
	})
	if err == nil {
		t.Fatal("expected error when postgres not configured, got nil")
	}
}

func TestPostgresCompactHandler_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	handler := postgresCompactHandler(s)
	_, err := handler(ctx, &dataretentionpb.CompactDataRetentionRequest{})
	if err == nil {
		t.Fatal("expected error when postgres not configured, got nil")
	}
}

func TestPostgresSpringCleanHandler_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	handler := postgresSpringCleanHandler(s)
	_, err := handler(ctx, &dataretentionpb.SpringCleanDataRetentionRequest{})
	if err == nil {
		t.Fatal("expected error when postgres not configured, got nil")
	}
}
