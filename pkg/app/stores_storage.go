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

// startStoresStorageTraits announces the storage trait for each configured store
// and starts polling goroutines. Returns an undo func to clean up.
func startStoresStorageTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, cfg *stores.Config, logger *zap.Logger) node.Undo {
	var undos []node.Undo

	if cfg != nil && cfg.DataDir != "" {
		undo := announceSqliteStorageTraits(ctx, n, nodeName, s, logger.Named("stores.sqlite"))
		undos = append(undos, undo)
	}

	if cfg != nil && cfg.Postgres != nil {
		undo := announcePostgresStorageTraits(ctx, n, nodeName, s, logger.Named("stores.postgres"))
		undos = append(undos, undo)
	}

	return node.UndoAll(undos...)
}

func announceSqliteStorageTraits(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, logger *zap.Logger) node.Undo {
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
		// poll once immediately
		updateSqliteStorageModel(ctx, s, model, logger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updateSqliteStorageModel(ctx, s, model, logger)
			}
		}
	}()

	return undo
}

func updateSqliteStorageModel(ctx context.Context, s *stores.Stores, model *storagepb.Model, logger *zap.Logger) {
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
	_, _ = model.SetStorage(&gen.Storage{
		Bytes: &gen.StorageBytes{Used: &used},
		Items: &gen.StorageItems{Used: &usedItems},
	})
}

func sqliteAdminHandler(ctx context.Context, s *stores.Stores, logger *zap.Logger) storagepb.AdminHandler {
	return func(reqCtx context.Context, action gen.StorageAdminAction) (*gen.PerformStorageAdminResponse, error) {
		db, err := s.SqliteHistory(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get sqlite history store: %v", err)
		}

		switch action {
		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR:
			deleted, err := db.Clear(reqCtx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "clear failed: %v", err)
			}
			deletedU := uint64(deleted)
			return &gen.PerformStorageAdminResponse{
				FreedItems: &gen.StorageItems{Used: &deletedU},
			}, nil

		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD:
			cutoff := time.Now().Add(-30 * 24 * time.Hour)
			deleted, err := db.TrimTime(reqCtx, "", cutoff)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "delete_old failed: %v", err)
			}
			deletedU := uint64(deleted)
			return &gen.PerformStorageAdminResponse{
				FreedItems: &gen.StorageItems{Used: &deletedU},
			}, nil

		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupported action: %v", action)
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
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		// poll once immediately
		updatePostgresStorageModel(ctx, s, model, logger)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updatePostgresStorageModel(ctx, s, model, logger)
			}
		}
	}()

	return undo
}

func updatePostgresStorageModel(ctx context.Context, s *stores.Stores, model *storagepb.Model, logger *zap.Logger) {
	r, _, _, err := s.Postgres()
	if err != nil {
		logger.Warn("failed to get postgres store", zap.Error(err))
		return
	}

	var sizeBytes int64
	err = r.QueryRow(ctx, "SELECT pg_total_relation_size('history')").Scan(&sizeBytes)
	if err != nil {
		logger.Warn("failed to query postgres history size", zap.Error(err))
		return
	}

	var rowCount int64
	err = r.QueryRow(ctx, "SELECT reltuples::bigint FROM pg_class WHERE relname = 'history'").Scan(&rowCount)
	if err != nil {
		logger.Warn("failed to query postgres history row count", zap.Error(err))
		return
	}

	used := uint64(sizeBytes)
	usedItems := uint64(rowCount)
	_, _ = model.SetStorage(&gen.Storage{
		Bytes: &gen.StorageBytes{Used: &used},
		Items: &gen.StorageItems{Used: &usedItems},
	})
}

func postgresAdminHandler(s *stores.Stores, logger *zap.Logger) storagepb.AdminHandler {
	return func(ctx context.Context, action gen.StorageAdminAction) (*gen.PerformStorageAdminResponse, error) {
		_, w, _, err := s.Postgres()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get postgres store: %v", err)
		}

		switch action {
		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_VACUUM:
			_, err = w.Exec(ctx, "VACUUM ANALYZE history")
			if err != nil {
				return nil, status.Errorf(codes.Internal, "vacuum failed: %v", err)
			}
			return &gen.PerformStorageAdminResponse{}, nil

		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_DELETE_OLD:
			cutoff := time.Now().Add(-30 * 24 * time.Hour)
			tag, err := w.Exec(ctx, "DELETE FROM history WHERE create_time < $1", cutoff)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "delete_old failed: %v", err)
			}
			deletedU := uint64(tag.RowsAffected())
			return &gen.PerformStorageAdminResponse{
				FreedItems: &gen.StorageItems{Used: &deletedU},
			}, nil

		case gen.StorageAdminAction_STORAGE_ADMIN_ACTION_CLEAR:
			tag, err := w.Exec(ctx, "DELETE FROM history")
			if err != nil {
				return nil, status.Errorf(codes.Internal, "clear failed: %v", err)
			}
			deletedU := uint64(tag.RowsAffected())
			return &gen.PerformStorageAdminResponse{
				FreedItems: &gen.StorageItems{Used: &deletedU},
			}, nil

		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupported action: %v", action)
		}
	}
}
