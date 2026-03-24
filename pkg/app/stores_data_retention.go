package app

import (
	"context"
	"path"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	server := dataretentionpb.NewModelServer(model,
		dataretentionpb.WithClearHandler(sqliteClearHandler(s)),
		dataretentionpb.WithDeleteOldHandler(sqliteDeleteOldHandler(s)),
		dataretentionpb.WithDataRetentionSupport(&dataretentionpb.DataRetentionSupport{
			CanClear:     true,
			CanDeleteOld: true,
			ItemName:     "record",
		}),
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
	if cap, util, ok := sqliteDiskCapacity(dataDir, used); ok {
		bytesMsg.Capacity = &cap
		bytesMsg.Utilization = &util
	}

	_, _ = model.SetDataRetention(&dataretentionpb.DataRetention{
		Bytes: bytesMsg,
		Items: &dataretentionpb.DataRetentionItems{Used: &usedItems},
	})
}

func sqliteClearHandler(s *stores.Stores) dataretentionpb.ClearHandler {
	return func(ctx context.Context, _ *dataretentionpb.ClearDataRetentionRequest) (*dataretentionpb.ClearDataRetentionResponse, error) {
		db, err := s.SqliteHistory(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get sqlite history store: %v", err)
		}
		deleted, err := db.Clear(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "clear failed: %v", err)
		}
		deletedU := uint64(deleted)
		return &dataretentionpb.ClearDataRetentionResponse{FreedItemCount: &deletedU}, nil
	}
}

func sqliteDeleteOldHandler(s *stores.Stores) dataretentionpb.DeleteOldHandler {
	return func(ctx context.Context, req *dataretentionpb.DeleteOldDataRetentionRequest) (*dataretentionpb.DeleteOldDataRetentionResponse, error) {
		if req.RetentionPeriod == nil {
			return nil, status.Error(codes.InvalidArgument, "retention_period must be set to a positive value")
		}
		d := req.RetentionPeriod.AsDuration()
		if d <= 0 {
			return nil, status.Error(codes.InvalidArgument, "retention_period must be positive")
		}
		db, err := s.SqliteHistory(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get sqlite history store: %v", err)
		}
		cutoff := time.Now().Add(-d)
		deleted, err := db.TrimTime(ctx, "", cutoff)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "delete_old failed: %v", err)
		}
		deletedU := uint64(deleted)
		return &dataretentionpb.DeleteOldDataRetentionResponse{FreedItemCount: &deletedU}, nil
	}
}

func announcePostgresDataRetentionTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, logger *zap.Logger) node.Undo {
	model := dataretentionpb.NewModel()
	server := dataretentionpb.NewModelServer(model,
		dataretentionpb.WithClearHandler(postgresClearHandler(s)),
		dataretentionpb.WithDeleteOldHandler(postgresDeleteOldHandler(s)),
		dataretentionpb.WithCompactHandler(postgresCompactHandler(s)),
		dataretentionpb.WithSpringCleanHandler(postgresSpringCleanHandler(s)),
		dataretentionpb.WithDataRetentionSupport(&dataretentionpb.DataRetentionSupport{
			CanClear:       true,
			CanDeleteOld:   true,
			CanCompact:     true,
			CanSpringClean: true,
			ItemName:       "row",
		}),
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
		_, _ = model.SetDataRetention(&dataretentionpb.DataRetention{})
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

func postgresClearHandler(s *stores.Stores) dataretentionpb.ClearHandler {
	return func(ctx context.Context, _ *dataretentionpb.ClearDataRetentionRequest) (*dataretentionpb.ClearDataRetentionResponse, error) {
		_, w, _, err := s.Postgres()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get postgres store: %v", err)
		}
		tag, err := w.Exec(ctx, "DELETE FROM history")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "clear failed: %v", err)
		}
		deletedU := uint64(tag.RowsAffected())
		return &dataretentionpb.ClearDataRetentionResponse{FreedItemCount: &deletedU}, nil
	}
}

func postgresDeleteOldHandler(s *stores.Stores) dataretentionpb.DeleteOldHandler {
	return func(ctx context.Context, req *dataretentionpb.DeleteOldDataRetentionRequest) (*dataretentionpb.DeleteOldDataRetentionResponse, error) {
		if req.RetentionPeriod == nil {
			return nil, status.Error(codes.InvalidArgument, "retention_period must be set to a positive value")
		}
		d := req.RetentionPeriod.AsDuration()
		if d <= 0 {
			return nil, status.Error(codes.InvalidArgument, "retention_period must be positive")
		}
		_, w, _, err := s.Postgres()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get postgres store: %v", err)
		}
		cutoff := time.Now().Add(-d)
		tag, err := w.Exec(ctx, "DELETE FROM history WHERE create_time < $1", cutoff)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "delete_old failed: %v", err)
		}
		deletedU := uint64(tag.RowsAffected())
		return &dataretentionpb.DeleteOldDataRetentionResponse{FreedItemCount: &deletedU}, nil
	}
}

func postgresCompactHandler(s *stores.Stores) dataretentionpb.CompactHandler {
	return func(ctx context.Context, _ *dataretentionpb.CompactDataRetentionRequest) (*dataretentionpb.CompactDataRetentionResponse, error) {
		_, w, _, err := s.Postgres()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get postgres store: %v", err)
		}
		_, err = w.Exec(ctx, "VACUUM ANALYZE history")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "compact failed: %v", err)
		}
		return &dataretentionpb.CompactDataRetentionResponse{}, nil
	}
}

func postgresSpringCleanHandler(s *stores.Stores) dataretentionpb.SpringCleanHandler {
	return func(ctx context.Context, req *dataretentionpb.SpringCleanDataRetentionRequest) (*dataretentionpb.SpringCleanDataRetentionResponse, error) {
		_, w, _, err := s.Postgres()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get postgres store: %v", err)
		}

		resp := &dataretentionpb.SpringCleanDataRetentionResponse{}

		if req.RetentionPeriod != nil {
			d := req.RetentionPeriod.AsDuration()
			if d <= 0 {
				return nil, status.Error(codes.InvalidArgument, "retention_period must be positive")
			}
			cutoff := time.Now().Add(-d)
			tag, err := w.Exec(ctx, "DELETE FROM history WHERE create_time < $1", cutoff)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "delete_old failed: %v", err)
			}
			deletedU := uint64(tag.RowsAffected())
			resp.FreedItemCount = &deletedU
		}

		_, err = w.Exec(ctx, "VACUUM ANALYZE history")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "compact failed: %v", err)
		}

		return resp, nil
	}
}
