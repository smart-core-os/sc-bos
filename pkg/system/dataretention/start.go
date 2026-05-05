// Package dataretention announces the DataRetention trait for each configured store
// and manages the associated polling goroutines.
package dataretention

import (
	"context"
	"path"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
)

// Start announces the DataRetention trait for each configured store and starts
// subscriber-aware polling goroutines. Returns an undo func that removes all announcements.
//
// Each backend is announced as an independent device:
//   - {nodeName}/stores/history  – SQLite history store
//   - {nodeName}/stores/postgres – Postgres history store
//
// These devices must not be aggregated (e.g. summing bytes.used is not meaningful
// across heterogeneous backends).
func Start(ctx context.Context, n *node.Node, nodeName string, s *stores.Stores, cfg *stores.Config, logger *zap.Logger) node.Undo {
	var undos []node.Undo

	if cfg != nil && cfg.DataDir != "" {
		name := path.Join(nodeName, "stores/history")
		undo := announceSqlite(ctx, n, name, s, cfg.DataDir, logger.Named("stores.sqlite"))
		undos = append(undos, undo)
	}

	if cfg != nil && cfg.Postgres != nil {
		name := path.Join(nodeName, "stores/postgres")
		undo := announcePostgres(ctx, n, name, s, logger.Named("stores.postgres"))
		undos = append(undos, undo)
	}

	return node.UndoAll(undos...)
}
