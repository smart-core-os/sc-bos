// Package history provides an automation that pulls data from a trait and inserts them into store.
// The automation announces a history api to allow API retrieval of these records filtered by time period.
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
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/historypb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/transport"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/unlockablepb"
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/history/apistore"
	"github.com/smart-core-os/sc-bos/pkg/history/boltstore"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

var Factory = auto.FactoryFunc(NewAutomation)

func NewAutomation(services auto.Services) service.Lifecycle {
	a := &automation{
		clients:   services.Node,
		announcer: node.NewReplaceAnnouncer(services.Node),
		logger:    services.Logger.Named("history"),

		db:     services.Database,
		stores: services.Stores,

		cohortManagerName: "", // use the default
		cohortManager:     services.CohortManager,
	}
	a.Service = service.New(
		service.MonoApply(a.applyConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", a.logger)
		})),
	)
	return a
}

type automation struct {
	*service.Service[config.Root]
	clients   node.ClientConner
	announcer *node.ReplaceAnnouncer
	logger    *zap.Logger

	db     *bolthold.Store
	stores *stores.Stores

	cohortManagerName string
	cohortManager     node.Remote
}

func (a *automation) applyConfig(ctx context.Context, cfg config.Root) error {
	a.logger.Debug("applying config", zap.Any("storageType", cfg.Storage.Type), zap.Any("trait", cfg.Source.Trait))
	// work out where we're storing the history
	var store history.Store
	switch cfg.Storage.Type {
	case "postgres":
		var pool *pgxpool.Pool
		var err error
		if cfg.Storage.ConnectConfig.IsZero() {
			// use admin pool (for now) as we know it will support create table operations
			// todo: update store to support r, w, admin pools
			_, _, pool, err = a.stores.Postgres()
		} else {
			pool, err = pgxutil.Connect(ctx, cfg.Storage.ConnectConfig)
		}
		if err != nil {
			return err
		}
		opts := []pgxstore.Option{
			pgxstore.WithLogger(a.logger),
		}
		if ttl := cfg.Storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, pgxstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, pgxstore.WithMaxCount(ttl.MaxCount))
			}
		}
		store, err = pgxstore.SetupStoreFromPool(ctx, cfg.Source.SourceName(), pool, opts...)
		if err != nil {
			return err
		}
	case "memory":
		var opts []memstore.Option
		if ttl := cfg.Storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, memstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, memstore.WithMaxCount(ttl.MaxCount))
			}
		}
		store = memstore.New(opts...)
	case "api":
		if cfg.Storage.TTL != nil {
			a.logger.Warn("storage.ttl ignored when storage.type is \"api\"")
		}
		name := cfg.Storage.Name
		if name == "" {
			return errors.New("storage.name missing, must exist when storage.type is \"api\"")
		}
		client := gen.NewHistoryAdminApiClient(a.clients.ClientConn())
		store = apistore.New(client, name, cfg.Source.SourceName())
	case "hub":
		if cfg.Storage.TTL != nil {
			a.logger.Warn("storage.ttl ignored when storage.type is \"hub\"")
		}
		conn, err := a.cohortManager.Connect(ctx)
		if err != nil {
			return err
		}
		client := gen.NewHistoryAdminApiClient(conn)
		store = apistore.New(client, a.cohortManagerName, cfg.Source.SourceName())
	case "bolt":
		var err error
		opts := []boltstore.Option{
			boltstore.WithLogger(a.logger),
		}
		if ttl := cfg.Storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, boltstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, boltstore.WithMaxCount(ttl.MaxCount))
			}
		}
		store, err = boltstore.NewFromDb(ctx, a.db, cfg.Source.SourceName(), opts...)
		if err != nil {
			return err
		}
	case "sqlite":
		db, err := a.stores.SqliteHistory(ctx)
		if err != nil {
			return err
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
		store = db.OpenStore(cfg.Source.SourceName(), opts...)
	default:
		return fmt.Errorf("unsupported storage type %s", cfg.Storage.Type)
	}

	// work out where we're getting the records from
	var serverClient wrap.ServiceUnwrapper
	payloads := make(chan []byte)
	var collect collector
	switch cfg.Source.Trait {
	case trait.AirQualitySensor:
		serverClient = gen.WrapAirQualitySensorHistory(historypb.NewAirQualitySensorServer(store))
		collect = a.collectAirQualityChanges
	case trait.AirTemperature:
		serverClient = gen.WrapAirTemperatureHistory(historypb.NewAirTemperatureServer(store))
		collect = a.collectAirTemperatureChanges
	case trait.Electric:
		serverClient = gen.WrapElectricHistory(historypb.NewElectricServer(store))
		collect = a.collectElectricDemandChanges
	case trait.EnterLeaveSensor:
		serverClient = gen.WrapEnterLeaveHistory(historypb.NewEnterLeaveSensorServer(store))
		collect = a.collectEnterLeaveEventChanges
	case unlockablepb.TraitName:
		serverClient = gen.WrapUnlockableAPI(historypb.NewUnlockableServer(store))
		collect = a.collectLockerChanges
	case meter.TraitName:
		serverClient = gen.WrapMeterHistory(historypb.NewMeterServer(store))
		collect = a.collectMeterReadingChanges
	case trait.OccupancySensor:
		serverClient = gen.WrapOccupancySensorHistory(historypb.NewOccupancySensorServer(store))
		collect = a.collectOccupancyChanges
	case statuspb.TraitName:
		serverClient = gen.WrapStatusHistory(historypb.NewStatusServer(store))
		collect = a.collectCurrentStatusChanges
	case transport.TraitName:
		serverClient = gen.WrapTransportHistory(historypb.NewTransportServer(store))
		collect = a.collectTransportChanges
	case soundsensorpb.TraitName:
		serverClient = gen.WrapSoundSensorHistory(historypb.NewSoundSensorServer(store))
		collect = a.collectSoundSensorChanges
	default:
		return fmt.Errorf("unsupported trait %s", cfg.Source.Trait)
	}

	// each time the source emits, we append it to the store
	go func() {
		defer close(payloads)

		for {
			select {
			case <-ctx.Done():
				return
			case payload := <-payloads:
				_, err := store.Append(ctx, payload)
				if err != nil {
					a.logger.Warn("storage failed", zap.Error(err))
				}
			}
		}
	}()

	announce := a.announcer.Replace(ctx)
	// announce the trait too to ensure its services get added to the router before the collect routine starts
	announce.Announce(cfg.Source.Name, node.HasClient(serverClient), node.HasTrait(cfg.Source.Trait))

	go collect(ctx, *cfg.Source, payloads)

	return nil
}
