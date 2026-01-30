package alerts

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/alerts/config"
	"github.com/smart-core-os/sc-bos/pkg/system/alerts/hubalerts"
	"github.com/smart-core-os/sc-bos/pkg/system/alerts/pgxalerts"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

var Factory factory

type factory struct{}

func (_ factory) New(services system.Services) service.Lifecycle {
	return NewSystem(services)
}

func NewSystem(services system.Services) *System {
	logger := services.Logger.Named("alerts")
	s := &System{
		name:      services.Node.Name(),
		announcer: node.NewReplaceAnnouncer(services.Node),

		cohortManagerName: "", // use the default
		cohortManager:     services.CohortManager,

		stores: services.Stores,
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

	cohortManagerName string
	cohortManager     node.Remote

	stores *stores.Stores
}

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	// using AnnounceContext only makes when using MonoApply, which we are in NewSystem
	announcer := s.announcer.Replace(ctx)

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

		server, err := pgxalerts.NewServerFromPool(ctx, pool)
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}

		announcer.Announce(s.name, node.HasClient(
			alertpb.WrapApi(server),
			alertpb.WrapAdminApi(server),
		))
	case config.StorageTypeHub:
		server := hubalerts.NewServer("", s.name, s.cohortManager)
		announcer.Announce(s.name, node.HasClient(
			alertpb.WrapApi(server),
			alertpb.WrapAdminApi(server),
		))
	default:
		return fmt.Errorf("unsuported storage type %s", cfg.Storage.Type)
	}

	return nil
}
