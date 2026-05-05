package dataretention

import (
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
)

func TestUpdatePostgresModel_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	model := dataretentionpb.NewModel()
	updatePostgresModel(ctx, s, model, zap.NewNop())

	got, err := model.GetDataRetention()
	if err != nil {
		t.Fatalf("GetDataRetention: %v", err)
	}
	if got.Bytes != nil || got.Items != nil {
		t.Errorf("expected empty data retention model when postgres not configured, got %v", got)
	}
}

func TestPostgresBackend_Purge_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	b := &postgresBackend{stores: s}
	_, err := b.Purge(ctx, nil)
	if err == nil {
		t.Fatal("expected error when postgres not configured, got nil")
	}
}

func TestPostgresBackend_Compact_NotConfigured(t *testing.T) {
	ctx := t.Context()
	s := stores.New(&stores.Config{DataDir: t.TempDir()})
	t.Cleanup(func() { _ = s.Close() })

	b := &postgresBackend{stores: s}
	err := b.Compact(ctx)
	if err == nil {
		t.Fatal("expected error when postgres not configured, got nil")
	}
}
