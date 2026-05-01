package app

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
)

// startStoresDataRetentionTraits announces the data retention trait for each configured store
// and starts polling goroutines. Returns an undo func to clean up.
func startStoresDataRetentionTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, cfg *stores.Config, logger *zap.Logger) node.Undo {
	var undos []node.Undo

	// Each backend is announced as an independent DataRetention trait device.
	// {nodeName}/stores/history and {nodeName}/stores/postgres measure separate backends
	// and must not be aggregated together (e.g. summing bytes.used would not be meaningful).
	if cfg != nil && cfg.DataDir != "" {
		undo := announceSqliteDataRetentionTraits(ctx, n, nodeName, s, cfg.DataDir, logger.Named("stores.sqlite"))
		undos = append(undos, undo)
	}

	if cfg != nil && cfg.Postgres != nil {
		undo := announcePostgresDataRetentionTraits(ctx, n, nodeName, s, logger.Named("stores.postgres"))
		undos = append(undos, undo)
	}

	return node.UndoAll(undos...)
}

func announceSqliteDataRetentionTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, dataDir string, logger *zap.Logger) node.Undo {
	model := dataretentionpb.NewModel()
	server := dataretentionpb.NewModelServer(model, &sqliteBackend{stores: s},
		dataretentionpb.WithItemName("record"),
	)

	name := path.Join(nodeName, "stores/history")
	undo := n.Announce(name, node.HasTrait(dataretentionpb.TraitName, node.WithClients(dataretentionpb.WrapApi(server))))

	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		// poll once immediately so GetDataRetention returns a value without waiting 30s
		updateSqliteDataRetentionModel(ctx, s, model, dataDir, logger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updateSqliteDataRetentionModel(ctx, s, model, dataDir, logger)
			}
		}
	}()

	return undo
}

func updateSqliteDataRetentionModel(ctx context.Context, s *stores.Stores, model *dataretentionpb.Model, dataDir string, logger *zap.Logger) {
	db, err := s.SqliteHistory(ctx)
	if err != nil {
		logger.Warn("failed to get sqlite history store", zap.Error(err))
		return
	}

	sizeBytes, err := db.Size(ctx)
	if err != nil {
		logger.Warn("failed to get sqlite history size", zap.Error(err))
		return
	}

	count, err := db.TotalCount(ctx)
	if err != nil {
		logger.Warn("failed to get sqlite history count", zap.Error(err))
		return
	}

	used := uint64(sizeBytes)
	usedItems := uint64(count)

	bytesMsg := &dataretentionpb.DataRetentionBytes{Used: &used}
	if cap, _, ok := sqliteDiskCapacity(dataDir, used); ok {
		bytesMsg.Capacity = &cap
	}

	_, _ = model.SetDataRetention(&dataretentionpb.DataRetention{
		Bytes: bytesMsg,
		Items: &dataretentionpb.DataRetentionItems{Used: &usedItems},
	})
}

// sqliteBackend implements dataretentionpb.Backend for the SQLite history store.
type sqliteBackend struct {
	stores *stores.Stores
}

func (b *sqliteBackend) Purge(ctx context.Context, before *time.Time) (uint64, error) {
	db, err := b.stores.SqliteHistory(ctx)
	if err != nil {
		return 0, fmt.Errorf("sqlite history: %w", err)
	}
	if before == nil {
		deleted, err := db.Clear(ctx)
		if err != nil {
			return 0, err
		}
		return uint64(deleted), nil
	}
	deleted, err := db.TrimTime(ctx, "", *before)
	if err != nil {
		return 0, err
	}
	return uint64(deleted), nil
}

func announcePostgresDataRetentionTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, logger *zap.Logger) node.Undo {
	model := dataretentionpb.NewModel()
	server := dataretentionpb.NewModelServer(model, &postgresBackend{stores: s},
		dataretentionpb.WithItemName("row"),
	)

	name := path.Join(nodeName, "stores/postgres")
	undo := n.Announce(name, node.HasTrait(dataretentionpb.TraitName, node.WithClients(dataretentionpb.WrapApi(server))))

	go func() {
		var cache postgresRowCountCache
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		// poll once immediately so GetDataRetention returns a value without waiting 30s
		updatePostgresDataRetentionModel(ctx, s, model, &cache, logger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updatePostgresDataRetentionModel(ctx, s, model, &cache, logger)
			}
		}
	}()

	return undo
}

// postgresRowCountCache caches the result of COUNT(*) FROM history.
// COUNT(*) is expensive on large tables; we only need to refresh it hourly.
// Not safe for concurrent use — owned by a single goroutine.
type postgresRowCountCache struct {
	count     int64
	expiresAt time.Time
}

func (c *postgresRowCountCache) get() (int64, bool) {
	if time.Now().Before(c.expiresAt) {
		return c.count, true
	}
	return 0, false
}

func (c *postgresRowCountCache) set(count int64) {
	c.count = count
	c.expiresAt = time.Now().Add(1 * time.Hour)
}

func updatePostgresDataRetentionModel(ctx context.Context, s *stores.Stores, model *dataretentionpb.Model, cache *postgresRowCountCache, logger *zap.Logger) {
	r, _, _, err := s.Postgres()
	if err != nil {
		logger.Warn("failed to get postgres store", zap.Error(err))
		return
	}

	retention := &dataretentionpb.DataRetention{}

	var sizeBytes int64
	if err = r.QueryRow(ctx, "SELECT pg_total_relation_size('history')").Scan(&sizeBytes); err != nil {
		logger.Warn("failed to query postgres history size", zap.Error(err))
	} else {
		used := uint64(sizeBytes)
		retention.Bytes = &dataretentionpb.DataRetentionBytes{Used: &used}
	}

	// COUNT(*) is expensive; use the cached value if still fresh.
	if n, ok := cache.get(); ok {
		used := uint64(n)
		retention.Items = &dataretentionpb.DataRetentionItems{Used: &used}
	} else {
		var n int64
		if err = r.QueryRow(ctx, "SELECT COUNT(*) FROM history").Scan(&n); err != nil {
			logger.Warn("failed to query postgres history row count", zap.Error(err))
		} else {
			cache.set(n)
			used := uint64(n)
			retention.Items = &dataretentionpb.DataRetentionItems{Used: &used}
		}
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
	return postgresDelete(ctx, w, before)
}

func (b *postgresBackend) Compact(ctx context.Context) error {
	_, w, _, err := b.stores.Postgres()
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	_, err = w.Exec(ctx, "VACUUM ANALYZE history")
	return err
}

// postgresDelete deletes history rows. If before is nil, all rows are removed;
// otherwise only rows with create_time < *before.
func postgresDelete(ctx context.Context, w *pgxpool.Pool, before *time.Time) (uint64, error) {
	var sql string
	var args []any
	if before == nil {
		sql = "DELETE FROM history"
	} else {
		sql = "DELETE FROM history WHERE create_time < $1"
		args = append(args, *before)
	}
	tag, err := w.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return uint64(tag.RowsAffected()), nil
}
