package publications

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/publications/config"
	"github.com/smart-core-os/sc-bos/pkg/system/publications/pgxpublications"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/publicationpb"
)

var Factory factory

type factory struct{}

func (_ factory) New(services system.Services) service.Lifecycle {
	return NewSystem(services)
}

func NewSystem(services system.Services) *System {
	s := &System{
		logger:    services.Logger.Named("publications"),
		name:      services.Node.Name(),
		announcer: node.NewReplaceAnnouncer(services.Node),
		stores:    services.Stores,
	}
	s.Service = service.New(
		service.MonoApply(s.applyConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", s.logger)
		})),
	)
	return s
}

type System struct {
	*service.Service[config.Root]
	logger *zap.Logger

	name      string
	announcer *node.ReplaceAnnouncer

	stores *stores.Stores
}

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	if cfg.Storage == nil {
		return errors.New("no storage")
	}
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

		server, err := pgxpublications.NewServerFromPool(ctx, pool, pgxpublications.WithLogger(s.logger))
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}

		// Note, ctx in cancelled each time config is updated (and on stop) because we use MonoApply in NewSystem
		announcer := s.announcer.Replace(ctx)
		announcer.Announce(s.name, node.HasTrait(trait.Publication, node.WithClients(publicationpb.WrapApi(server))))
	default:
		return fmt.Errorf("unsuported storage type %s", cfg.Storage.Type)
	}

	return nil
}
