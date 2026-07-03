package dataretention

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func announcePostgres(ctx context.Context, n *node.Node, name string, s *stores.Stores, checks *healthpb.Checks, highPct float32, maxSizeBytes uint64, logger *zap.Logger) node.Undo {
	model := dataretentionpb.NewModel()
	server := dataretentionpb.NewModelServer(model, &postgresBackend{stores: s},
		dataretentionpb.WithItemName("row"),
	)

	undo := n.Announce(name, node.HasTrait(dataretentionpb.TraitName, node.WithClients(dataretentionpb.WrapApi(server))))

	health := &storageHealth{checks: checks, name: name, highPct: highPct, logger: logger}

	// Postgres only knows its capacity when a max size is configured. Without it there is
	// no storage HealthCheck, so retain the subscriber-aware idling; with it, the check is
	// an always-on consumer so we poll unconditionally. full is ignored: every figure the
	// update queries is cheap (ApproxCount reads pg_stat, no scan) or needed by the check.
	pollUndo := startPolling(ctx, model, maxSizeBytes > 0, health, func(ctx context.Context, _ bool) {
		updatePostgresModel(ctx, s, model, health, maxSizeBytes, logger)
	})

	return node.UndoAll(undo, pollUndo)
}

func updatePostgresModel(ctx context.Context, s *stores.Stores, model *dataretentionpb.Model, health *storageHealth, maxSizeBytes uint64, logger *zap.Logger) {
	r, _, _, err := s.Postgres()
	if err != nil {
		logger.Warn("failed to get postgres store", zap.Error(err))
		return
	}

	retention := &dataretentionpb.DataRetention{}

	sizeBytes, err := pgxstore.TotalSize(ctx, r)
	if err != nil {
		logger.Warn("failed to query postgres history size", zap.Error(err))
	} else {
		used := uint64(sizeBytes)
		bytesMsg := &dataretentionpb.DataRetentionBytes{Used: &used}
		if maxSizeBytes > 0 {
			capacity := maxSizeBytes
			bytesMsg.Capacity = &capacity
			health.update(ctx, float32(used)/float32(maxSizeBytes)*100)
		}
		retention.Bytes = bytesMsg
	}

	// ApproxCount reads n_live_tup from pg_stat_user_tables — O(1), no sequential scan.
	n, err := pgxstore.ApproxCount(ctx, r)
	if err != nil {
		logger.Warn("failed to query postgres history row count", zap.Error(err))
	} else {
		used := uint64(n)
		retention.Items = &dataretentionpb.DataRetentionItems{Used: &used}
	}

	_, _ = model.SetDataRetention(retention)
}

// postgresBackend implements dataretentionpb.Backend and dataretentionpb.Compacter
// for the Postgres history store.
type postgresBackend struct {
	stores *stores.Stores
}

func (b *postgresBackend) Purge(ctx context.Context, before *time.Time) (uint64, error) {
	_, w, _, err := b.stores.Postgres()
	if err != nil {
		return 0, fmt.Errorf("postgres: %w", err)
	}
	return pgxstore.DeleteAll(ctx, w, before)
}

func (b *postgresBackend) Compact(ctx context.Context) error {
	// VACUUM requires table ownership / the MAINTAIN privilege, so use the admin pool.
	_, _, admin, err := b.stores.Postgres()
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	return pgxstore.Vacuum(ctx, admin)
}
