package tenants

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/tenantpb"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/tenants/config"
	"github.com/smart-core-os/sc-bos/pkg/system/tenants/pgxtenants"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

var Factory = factory{}

type factory struct{}

func (f factory) New(services system.Services) service.Lifecycle {
	return NewSystem(services)
}

func NewSystem(services system.Services) *System {
	s := &System{
		node:    services.Node,
		hubNode: services.CohortManager,
		stores:  services.Stores,
		logger:  services.Logger.Named("tenants"),
	}
	s.Service = service.New(
		s.applyConfig,
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", s.logger)
		})),
	)
	return s
}

type System struct {
	*service.Service[config.Root]
	undos   []node.Undo
	node    *node.Node
	hubNode node.Remote
	stores  *stores.Stores
	logger  *zap.Logger
}

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	// clear out announcements from previous runs
	for _, u := range s.undos {
		u()
	}
	s.undos = nil

	if cfg.Storage == nil {
		return errors.New("no storage")
	}
	var srv *node.Service
	switch cfg.Storage.Type {
	case config.StorageTypeProxy:
		s.logger.Warn("proxy storage type is deprecated - use gateway to route requests to the hub instead")

		conn, err := s.hubNode.Connect(ctx)
		if err != nil {
			return err
		}

		srv, err = node.RegistryConnService(tenantpb.TenantApi_ServiceDesc, conn)
		if err != nil {
			return fmt.Errorf("can't create proxied TenantApi service: %w", err)
		}
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

		server, err := pgxtenants.NewServerFromPool(ctx, pool, pgxtenants.WithLogger(s.logger))
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}

		srv, err = node.RegistryService(tenantpb.TenantApi_ServiceDesc, server)
		if err != nil {
			return fmt.Errorf("can't create local TenantApi service: %w", err)
		}
	default:
		return fmt.Errorf("unsuported storage type %s", cfg.Storage.Type)
	}

	undo, err := s.node.AnnounceService(srv)
	s.undos = append(s.undos, undo)
	if err != nil {
		return fmt.Errorf("can't announce TenantApi service: %w", err)
	}

	return nil
}
