package dataretention

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func announceSqlite(ctx context.Context, n *node.Node, name string, s *stores.Stores, dataDir string, checks *healthpb.Checks, highPct float32, logger *zap.Logger) node.Undo {
	model := dataretentionpb.NewModel()
	server := dataretentionpb.NewModelServer(model, &sqliteBackend{stores: s},
		dataretentionpb.WithItemName("row"),
	)

	undo := n.Announce(name, node.HasTrait(dataretentionpb.TraitName, node.WithClients(dataretentionpb.WrapApi(server))))

	health := &storageHealth{checks: checks, name: name, highPct: highPct, logger: logger}

	// The SQLite store reports filesystem capacity (where available), so its storage
	// HealthCheck is an always-on consumer: poll unconditionally rather than idling until
	// a PullDataRetention subscriber connects.
	pollUndo := startPolling(ctx, model, true, health, func(ctx context.Context, full bool) {
		updateSqliteModel(ctx, s, model, dataDir, health, full, logger)
	})

	return node.UndoAll(undo, pollUndo)
}

func updateSqliteModel(ctx context.Context, s *stores.Stores, model *dataretentionpb.Model, dataDir string, health *storageHealth, full bool, logger *zap.Logger) {
	db, err := s.SqliteHistory(ctx)
	if err != nil {
		logger.Warn("failed to get sqlite history store", zap.Error(err))
		// Disk fullness comes from the filesystem, not the store: report it even when the
		// store query fails — a full disk may be why.
		reportStorageHealth(ctx, health, dataDir, 0)
		return
	}

	sizeBytes, err := db.Size(ctx)
	if err != nil {
		logger.Warn("failed to get sqlite history size", zap.Error(err))
		reportStorageHealth(ctx, health, dataDir, 0)
		return
	}
	used := uint64(sizeBytes)

	bytesMsg := &dataretentionpb.DataRetentionBytes{Used: &used}
	if capacity, otherUsed, available, ok := diskCapacity(dataDir, used); ok {
		bytesMsg.Capacity = &capacity
		bytesMsg.OtherUsed = &otherUsed
		bytesMsg.Available = &available
		if denom := used + otherUsed + available; denom > 0 {
			health.update(ctx, float32(used+otherUsed)/float32(denom)*100)
		}
	}

	// Counting rows is a full table scan in SQLite — too expensive to repeat on every
	// poll when nobody is watching, so without full the last known count is kept.
	items := lastItems(model)
	if full {
		if count, err := db.TotalCount(ctx); err != nil {
			logger.Warn("failed to get sqlite history count", zap.Error(err))
		} else {
			usedItems := uint64(count)
			items = &dataretentionpb.DataRetentionItems{Used: &usedItems}
		}
	}

	_, _ = model.SetDataRetention(&dataretentionpb.DataRetention{
		Bytes: bytesMsg,
		Items: items,
	})
}

// reportStorageHealth reads disk stats for dataDir and reports df-style disk fullness —
// (used+otherUsed)/(used+otherUsed+available), as a percentage — to the storage health check.
// It is used on the store-failure paths (where the DB size is unknown, so dbUsedBytes is 0)
// so a full disk still raises the check even when the store query fails; a full disk may be
// the cause. The ratio is invariant to how disk usage splits between this store and other
// data, so passing 0 for dbUsedBytes still yields the correct fullness.
func reportStorageHealth(ctx context.Context, health *storageHealth, dataDir string, dbUsedBytes uint64) {
	_, otherUsed, available, ok := diskCapacity(dataDir, dbUsedBytes)
	if !ok {
		return
	}
	if denom := dbUsedBytes + otherUsed + available; denom > 0 {
		health.update(ctx, float32(dbUsedBytes+otherUsed)/float32(denom)*100)
	}
}

// lastItems returns a copy of the items figures currently in the model, or nil if there are none.
func lastItems(model *dataretentionpb.Model) *dataretentionpb.DataRetentionItems {
	v, err := model.GetDataRetention()
	if err != nil || v.GetItems() == nil {
		return nil
	}
	return proto.Clone(v.GetItems()).(*dataretentionpb.DataRetentionItems)
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
