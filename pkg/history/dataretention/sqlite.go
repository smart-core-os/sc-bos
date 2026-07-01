package dataretention

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/dataretentionpb"
)

func announceSqlite(ctx context.Context, n *node.Node, name string, s *stores.Stores, dataDir string, logger *zap.Logger) node.Undo {
	model := dataretentionpb.NewModel()
	server := dataretentionpb.NewModelServer(model, &sqliteBackend{stores: s},
		dataretentionpb.WithItemName("row"),
	)

	undo := n.Announce(name, node.HasTrait(dataretentionpb.TraitName, node.WithClients(dataretentionpb.WrapApi(server))))

	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		// poll once immediately so GetDataRetention returns a value without waiting 30s
		updateSqliteModel(ctx, s, model, dataDir, logger)
		for {
			if !model.HasSubscribers() {
				select {
				case <-ctx.Done():
					return
				case <-model.WaitForSubscriber():
					updateSqliteModel(ctx, s, model, dataDir, logger)
					tick.Reset(30 * time.Second)
				}
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				updateSqliteModel(ctx, s, model, dataDir, logger)
			}
		}
	}()

	return undo
}

func updateSqliteModel(ctx context.Context, s *stores.Stores, model *dataretentionpb.Model, dataDir string, logger *zap.Logger) {
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
	if cap, otherUsed, available, ok := diskCapacity(dataDir, used); ok {
		bytesMsg.Capacity = &cap
		bytesMsg.OtherUsed = &otherUsed
		bytesMsg.Available = &available
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
