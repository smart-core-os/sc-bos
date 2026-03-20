package app

import (
	"context"
	"path"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/storagepb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen "github.com/smart-core-os/sc-bos/pkg/proto/storagepb"
)

// retentionFromRequest extracts the retention period from the request.
// Returns an error if not set or zero, to prevent accidental data deletion.
func retentionFromRequest(req *gen.PerformStorageAdminRequest) (time.Duration, error) {
	if req.RetentionPeriod != nil {
		if d := req.RetentionPeriod.AsDuration(); d > 0 {
			return d, nil
		}
	}
	return 0, status.Error(codes.InvalidArgument, "retention_period must be set to a positive value for DELETE_OLD")
}

// startStoresStorageTraits announces the storage trait for each configured store
// and starts polling goroutines. Returns an undo func to clean up.
func startStoresStorageTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, cfg *stores.Config, logger *zap.Logger) node.Undo {
	var undos []node.Undo

	// Each backend is announced as an independent Storage trait device.
	// {nodeName}/stores/history and {nodeName}/stores/postgres measure separate backends
	// and must not be aggregated together (e.g. summing bytes.used would not be meaningful).
	if cfg != nil && cfg.DataDir != "" {
		undo := announceSqliteStorageTraits(ctx, n, nodeName, s, cfg.DataDir, logger.Named("stores.sqlite"))
		undos = append(undos, undo)
	}

	if cfg != nil && cfg.Postgres != nil {
		undo := announcePostgresStorageTraits(ctx, n, nodeName, s, logger.Named("stores.postgres"))
		undos = append(undos, undo)
	}

	return node.UndoAll(undos...)
}

func announceSqliteStorageTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, dataDir string, logger *zap.Logger) node.Undo {
	model := storagepb.NewModel()
	server := storagepb.NewModelServer(model,
		storagepb.WithAdminHandler(sqliteAdminHandler(ctx, s, logger)),
		storagepb.WithStorageSupport(&gen.StorageSupport{
			SupportedActions: []gen.StorageAdminAction{
				gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR,
				gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD,
			},
			ItemName: "record",
		}),
	)

	name := path.Join(nodeName, "stores/history")
	undo := n.Announce(name, node.HasTrait(storagepb.TraitName, node.WithClients(gen.WrapApi(server))))

	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		// poll once immediately so GetStorage returns a value without waiting 30s
		updateSqliteStorageModel(ctx, s, model, dataDir, logger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updateSqliteStorageModel(ctx, s, model, dataDir, logger)
			}
		}
	}()

	return undo
}

func updateSqliteStorageModel(ctx context.Context, s *stores.Stores, model *storagepb.Model, dataDir string, logger *zap.Logger) {
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

	bytesMsg := &gen.StorageBytes{Used: &used}
	if cap, util, ok := sqliteDiskCapacity(dataDir, used); ok {
		bytesMsg.Capacity = &cap
		bytesMsg.Utilization = &util
	}

	_, _ = model.SetStorage(&gen.Storage{
		Bytes: bytesMsg,
		Items: &gen.StorageItems{Used: &usedItems},
	})
}


func sqliteAdminHandler(ctx context.Context, s *stores.Stores, logger *zap.Logger) storagepb.AdminHandler {
	return func(reqCtx context.Context, req *gen.PerformStorageAdminRequest) (*gen.PerformStorageAdminResponse, error) {
		db, err := s.SqliteHistory(reqCtx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get sqlite history store: %v", err)
		}

		switch req.Action {
		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR:
			deleted, err := db.Clear(reqCtx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "clear failed: %v", err)
			}
			deletedU := uint64(deleted)
			return &gen.PerformStorageAdminResponse{FreedItemCount: &deletedU}, nil

		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD:
			retention, err := retentionFromRequest(req)
			if err != nil {
				return nil, err
			}
			cutoff := time.Now().Add(-retention)
			deleted, err := db.TrimTime(reqCtx, "", cutoff)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "delete_old failed: %v", err)
			}
			deletedU := uint64(deleted)
			return &gen.PerformStorageAdminResponse{FreedItemCount: &deletedU}, nil

		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupported action: %v", req.Action)
		}
	}
}

func announcePostgresStorageTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, logger *zap.Logger) node.Undo {
	model := storagepb.NewModel()
	server := storagepb.NewModelServer(model,
		storagepb.WithAdminHandler(postgresAdminHandler(s, logger)),
		storagepb.WithStorageSupport(&gen.StorageSupport{
			SupportedActions: []gen.StorageAdminAction{
				gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR,
				gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD,
				gen.StorageAdminAction_STORAGE_ADMIN_ACTION_VACUUM,
			},
			ItemName: "row",
		}),
	)

	name := path.Join(nodeName, "stores/postgres")
	undo := n.Announce(name, node.HasTrait(storagepb.TraitName, node.WithClients(gen.WrapApi(server))))

	go func() {
		var cache postgresRowCountCache
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		// poll once immediately so GetStorage returns a value without waiting 30s
		updatePostgresStorageModel(ctx, s, model, &cache, logger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updatePostgresStorageModel(ctx, s, model, &cache, logger)
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

func updatePostgresStorageModel(ctx context.Context, s *stores.Stores, model *storagepb.Model, cache *postgresRowCountCache, logger *zap.Logger) {
	r, _, _, err := s.Postgres()
	if err != nil {
		logger.Warn("failed to get postgres store", zap.Error(err))
		_, _ = model.SetStorage(&gen.Storage{})
		return
	}

	storage := &gen.Storage{}

	var sizeBytes int64
	if err = r.QueryRow(ctx, "SELECT pg_total_relation_size('history')").Scan(&sizeBytes); err != nil {
		logger.Warn("failed to query postgres history size", zap.Error(err))
	} else {
		used := uint64(sizeBytes)
		storage.Bytes = &gen.StorageBytes{Used: &used}
	}

	// COUNT(*) is expensive; use the cached value if still fresh.
	if n, ok := cache.get(); ok {
		used := uint64(n)
		storage.Items = &gen.StorageItems{Used: &used}
	} else {
		var n int64
		if err = r.QueryRow(ctx, "SELECT COUNT(*) FROM history").Scan(&n); err != nil {
			logger.Warn("failed to query postgres history row count", zap.Error(err))
		} else {
			cache.set(n)
			used := uint64(n)
			storage.Items = &gen.StorageItems{Used: &used}
		}
	}

	_, _ = model.SetStorage(storage)
}

func postgresAdminHandler(s *stores.Stores, logger *zap.Logger) storagepb.AdminHandler {
	return func(ctx context.Context, req *gen.PerformStorageAdminRequest) (*gen.PerformStorageAdminResponse, error) {
		_, w, _, err := s.Postgres()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get postgres store: %v", err)
		}

		switch req.Action {
		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_VACUUM:
			_, err = w.Exec(ctx, "VACUUM ANALYZE history")
			if err != nil {
				return nil, status.Errorf(codes.Internal, "vacuum failed: %v", err)
			}
			return &gen.PerformStorageAdminResponse{}, nil

		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD:
			retention, err := retentionFromRequest(req)
			if err != nil {
				return nil, err
			}
			cutoff := time.Now().Add(-retention)
			tag, err := w.Exec(ctx, "DELETE FROM history WHERE create_time < $1", cutoff)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "delete_old failed: %v", err)
			}
			deletedU := uint64(tag.RowsAffected())
			return &gen.PerformStorageAdminResponse{FreedItemCount: &deletedU}, nil

		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR:
			tag, err := w.Exec(ctx, "DELETE FROM history")
			if err != nil {
				return nil, status.Errorf(codes.Internal, "clear failed: %v", err)
			}
			deletedU := uint64(tag.RowsAffected())
			return &gen.PerformStorageAdminResponse{FreedItemCount: &deletedU}, nil

		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupported action: %v", req.Action)
		}
	}
}
