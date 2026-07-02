// Package dataretention announces the DataRetention trait for each configured store
// and manages the associated polling goroutines.
package dataretention

import (
	"context"
	"path"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// defaultStorageHealthHighPercent is the disk utilisation percentage above which the
// storage HealthCheck reports HIGH, when none is configured.
const defaultStorageHealthHighPercent float32 = 90

// Start announces the DataRetention trait for each configured store and starts
// polling goroutines. Returns an undo func that removes all announcements.
//
// Each backend is announced as an independent device:
//   - {nodeName}/stores/history  – SQLite history store
//   - {nodeName}/stores/postgres – Postgres history store
//
// These devices must not be aggregated (e.g. summing bytes.used is not meaningful
// across heterogeneous backends).
//
// A storage utilisation HealthCheck is raised against a store device (using checks) once
// the store's capacity is known. The check reports HIGH when disk utilisation reaches
// highPct (see StorageHealthHighPercent). The SQLite store learns its capacity from the
// filesystem; the Postgres store has no inherent capacity, so its check is only raised when
// PostgresConfig.MaxSizeBytes is configured.
func Start(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, cfg *stores.Config, checks *healthpb.Checks, logger *zap.Logger) node.Undo {
	var undos []node.Undo
	if cfg == nil {
		return node.UndoAll(undos...)
	}

	highPct := defaultStorageHealthHighPercent
	if cfg.StorageHealthHighPercent > 0 {
		highPct = cfg.StorageHealthHighPercent
	}

	if cfg.DataDir != "" {
		name := path.Join(nodeName, "stores/history")
		undo := announceSqlite(ctx, n, name, s, cfg.DataDir, checks, highPct, logger.Named("stores.sqlite"))
		undos = append(undos, undo)
	}

	if cfg.Postgres != nil {
		name := path.Join(nodeName, "stores/postgres")
		undo := announcePostgres(ctx, n, name, s, checks, highPct, cfg.Postgres.MaxSizeBytes, logger.Named("stores.postgres"))
		undos = append(undos, undo)
	}

	return node.UndoAll(undos...)
}
