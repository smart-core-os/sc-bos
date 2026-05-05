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
)

func announcePostgres(ctx context.Context, n *node.Node, name string, s *stores.Stores, logger *zap.Logger) node.Undo {
	model := dataretentionpb.NewModel()
	server := dataretentionpb.NewModelServer(model, &postgresBackend{stores: s},
		dataretentionpb.WithItemName("row"),
	)

	undo := n.Announce(name, node.HasTrait(dataretentionpb.TraitName, node.WithClients(dataretentionpb.WrapApi(server))))

	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		// poll once immediately so GetDataRetention returns a value without waiting 30s
		updatePostgresModel(ctx, s, model, logger)
		for {
			if !model.HasSubscribers() {
				select {
				case <-ctx.Done():
					return
				case <-model.WaitForSubscriber():
					updatePostgresModel(ctx, s, model, logger)
					tick.Reset(30 * time.Second)
				}
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updatePostgresModel(ctx, s, model, logger)
			}
		}
	}()

	return undo
}

func updatePostgresModel(ctx context.Context, s *stores.Stores, model *dataretentionpb.Model, logger *zap.Logger) {
	r, _, _, err := s.Postgres()
	if err != nil {
		logger.Warn("failed to get postgres store", zap.Error(err))
		return
	}

	// Construct a minimal Store to access the new query methods.
	// We use the read pool; these queries are read-only.
	store := pgxstore.NewStoreFromPool("", r)

	retention := &dataretentionpb.DataRetention{}

	sizeBytes, err := store.TotalSize(ctx)
	if err != nil {
		logger.Warn("failed to query postgres history size", zap.Error(err))
	} else {
		used := uint64(sizeBytes)
		retention.Bytes = &dataretentionpb.DataRetentionBytes{Used: &used}
	}

	// ApproxCount reads n_live_tup from pg_stat_user_tables — O(1), no sequential scan.
	n, err := store.ApproxCount(ctx)
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
	store := pgxstore.NewStoreFromPool("", w)
	return store.Delete(ctx, before)
}

func (b *postgresBackend) Compact(ctx context.Context) error {
	_, w, _, err := b.stores.Postgres()
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	store := pgxstore.NewStoreFromPool("", w)
	return store.Vacuum(ctx)
}
