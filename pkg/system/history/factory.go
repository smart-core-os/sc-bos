// Package history provides an implementation of HistoryAdminApi backed by a history.Store.
// Enabling this system on a controller will allow history automations to store their records with us if configured to do so.
package history

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/history/boltstore"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/historypb"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/history/config"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

var Factory factory

type factory struct{}

func (_ factory) New(services system.Services) service.Lifecycle {
	return NewSystem(services)
}

func NewSystem(services system.Services) *System {
	logger := services.Logger.Named("history")
	s := &System{
		name:      services.Node.Name(),
		announcer: node.NewReplaceAnnouncer(services.Node),
		db:        services.Database,
		stores:    services.Stores,
		logger:    logger,
	}
	s.Service = service.New(
		service.MonoApply(s.applyConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", logger)
		})),
	)
	return s
}

type System struct {
	*service.Service[config.Root]
	name      string
	announcer *node.ReplaceAnnouncer
	db        *bolthold.Store
	stores    *stores.Stores

	logger *zap.Logger
}

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	// using AnnounceContext only makes when using MonoApply, which we are in NewSystem
	announcer := s.announcer.Replace(ctx)

	if cfg.Storage == nil {
		return errors.New("no storage")
	}

	var store func(string) history.Store

	switch cfg.Storage.Type {
	case config.StorageTypePostgres:
		var pool *pgxpool.Pool
		var err error
		if cfg.Storage.ConnectConfig.IsZero() {
			_, _, pool, err = s.stores.Postgres()
		} else {
			pool, err = pgxutil.Connect(ctx, cfg.Storage.ConnectConfig)
		}
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}

		if err := pgxstore.SetupDB(ctx, pool); err != nil {
			return fmt.Errorf("setup: %w", err)
		}

		opts := []pgxstore.Option{
			pgxstore.WithLogger(s.logger),
		}
		if ttl := cfg.Storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, pgxstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, pgxstore.WithMaxCount(ttl.MaxCount))
			}
		}
		store = func(source string) history.Store {
			return pgxstore.NewStoreFromPool(source, pool, opts...)
		}
	case config.StorageTypeBolt:
		storeCollection := make(map[string]history.Store)

		opts := []boltstore.Option{
			boltstore.WithLogger(s.logger),
		}
		if ttl := cfg.Storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, boltstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, boltstore.WithMaxCount(ttl.MaxCount))
			}
		}

		store = func(source string) history.Store {
			st, ok := storeCollection[source]
			if !ok {
				var err error
				st, err = boltstore.NewFromDb(ctx, s.db, source, opts...)
				if err != nil {
					s.logger.Error("failed to create bolt store", zap.Error(err))
				} else {
					storeCollection[source] = st
				}
			}
			return st
		}
	case config.StorageTypeSqlite:
		store = func(source string) history.Store {
			db, err := s.stores.SqliteHistory(ctx)
			if err != nil {
				s.logger.Error("failed to create sqlite store",
					zap.Error(err),
					zap.String("source", source),
				)
				return nil
			}

			var opts []sqlitestore.WriteOption
			if ttl := cfg.Storage.TTL; ttl != nil {
				if ttl.MaxAge.Duration > 0 {
					opts = append(opts, sqlitestore.WithMaxAge(ttl.MaxAge.Duration))
				}
				if ttl.MaxCount > 0 {
					opts = append(opts, sqlitestore.WithMaxCount(ttl.MaxCount))
				}
			}

			return db.OpenStore(source, opts...)
		}
	default:
		return fmt.Errorf("unsuported storage type %s", cfg.Storage.Type)
	}

	server := &storeServer{store: store}
	announcer.Announce(s.name, node.HasClient(historypb.WrapAdminApi(server)))

	return nil
}
