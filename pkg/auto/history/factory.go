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
	"github.com/smart-core-os/sc-bos/pkg/history"
	"github.com/smart-core-os/sc-bos/pkg/history/apistore"
	"github.com/smart-core-os/sc-bos/pkg/history/boltstore"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/history/pgxstore"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/allocationpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/historypb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/resourceusepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

var Factory = auto.FactoryFunc(NewAutomation)

func NewAutomation(services auto.Services) service.Lifecycle {
	a := &automation{
		clients:   services.Node,
		announcer: node.NewReplaceAnnouncer(services.Node),
		logger:    services.Logger.Named("history"),

		db:     services.Database,
		stores: services.Stores,

		devices: services.Devices,

		cohortManagerName: "", // use the default
		cohortManager:     services.CohortManager,
	}
	a.Service = service.New(
		service.MonoApply(a.routeConfig),
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

	devices devicespb.DevicesApiClient

	cohortManagerName string
	cohortManager     node.Remote
}

func (a *automation) applyConfig(ctx context.Context, cfg config.Root) error {
	a.logger.Debug("applying config", zap.Any("storageType", cfg.Storage.Type), zap.Any("trait", cfg.Source.Trait))

	store, err := a.createStore(ctx, *cfg.Source, cfg.Storage)
	if err != nil {
		return err
	}

	serverClient, collect, err := a.createCollector(store, cfg.Source.Trait)
	if err != nil {
		return err
	}

	payloads := make(chan []byte)

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

// routeConfig dispatches to applyConfigDevices when cfg.Source.Devices is set, otherwise falls back to applyConfig.
func (a *automation) routeConfig(ctx context.Context, cfg config.Root) error {
	if cfg.Source != nil && len(cfg.Source.Devices) > 0 {
		if a.devices == nil {
			return errors.New("source.devices requires a DevicesApi service to be configured")
		}
		return a.applyConfigDevices(ctx, cfg)
	}
	return a.applyConfig(ctx, cfg)
}

func (a *automation) applyConfigDevices(ctx context.Context, cfg config.Root) error {
	if cfg.Source == nil || cfg.Source.Trait == "" {
		return errors.New("source.trait is required when source.devices is set")
	}

	announce := a.announcer.Replace(ctx)

	type activeRecorder struct {
		cancel context.CancelFunc
		undo   node.Undo
	}
	active := map[string]activeRecorder{}

	defer func() {
		for _, rec := range active {
			rec.cancel()
			rec.undo()
		}
	}()

	stream, err := a.devices.PullDevices(ctx, &devicespb.PullDevicesRequest{
		Query: &devicespb.Device_Query{Conditions: cfg.Source.DevicesPb()},
	})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}
		for _, change := range resp.GetChanges() {
			name := change.GetName()
			switch change.GetType() {
			case typespb.ChangeType_ADD:
				if rec, ok := active[name]; ok {
					rec.cancel()
					rec.undo()
				}
				deviceCtx, cancel := context.WithCancel(ctx)
				undo, err := a.startDeviceRecording(deviceCtx, announce, name, cfg)
				if err != nil {
					a.logger.Error("failed to start recording", zap.String("device", name), zap.Error(err))
					cancel()
					continue
				}
				active[name] = activeRecorder{cancel: cancel, undo: undo}
			case typespb.ChangeType_REMOVE:
				if rec, ok := active[name]; ok {
					rec.cancel()
					rec.undo()
					delete(active, name)
				}
			default:
				// ignore updates, they should not affect history recording
			}
		}
	}
}

func (a *automation) startDeviceRecording(ctx context.Context, announce node.Announcer, deviceName string, cfg config.Root) (node.Undo, error) {
	src := *cfg.Source
	src.Name = deviceName

	store, err := a.createStore(ctx, src, cfg.Storage)
	if err != nil {
		return node.NilUndo, err
	}

	serverClient, collect, err := a.createCollector(store, cfg.Source.Trait)
	if err != nil {
		return node.NilUndo, err
	}

	payloads := make(chan []byte)

	go func() {
		defer close(payloads)
		for {
			select {
			case <-ctx.Done():
				return
			case payload := <-payloads:
				if _, err := store.Append(ctx, payload); err != nil {
					a.logger.Warn("storage failed", zap.String("device", deviceName), zap.Error(err))
				}
			}
		}
	}()

	undo := announce.Announce(deviceName, node.HasClient(serverClient), node.HasTrait(cfg.Source.Trait))
	go collect(ctx, src, payloads)

	return undo, nil
}

func (a *automation) createStore(ctx context.Context, src config.Source, storage *config.Storage) (history.Store, error) {
	switch storage.Type {
	case "postgres":
		var pool *pgxpool.Pool
		var err error
		if storage.ConnectConfig.IsZero() {
			_, _, pool, err = a.stores.Postgres()
		} else {
			pool, err = pgxutil.Connect(ctx, storage.ConnectConfig)
		}
		if err != nil {
			return nil, err
		}
		opts := []pgxstore.Option{pgxstore.WithLogger(a.logger)}
		if ttl := storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, pgxstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, pgxstore.WithMaxCount(ttl.MaxCount))
			}
		}
		return pgxstore.SetupStoreFromPool(ctx, src.SourceName(), pool, opts...)
	case "memory":
		var opts []memstore.Option
		if ttl := storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, memstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, memstore.WithMaxCount(ttl.MaxCount))
			}
		}
		return memstore.New(opts...), nil
	case "api":
		if storage.TTL != nil {
			a.logger.Warn("storage.ttl ignored when storage.type is \"api\"")
		}
		name := storage.Name
		if name == "" {
			return nil, errors.New("storage.name missing, must exist when storage.type is \"api\"")
		}
		client := historypb.NewHistoryAdminApiClient(a.clients.ClientConn())
		return apistore.New(client, name, src.SourceName()), nil
	case "hub":
		if storage.TTL != nil {
			a.logger.Warn("storage.ttl ignored when storage.type is \"hub\"")
		}
		conn, err := a.cohortManager.Connect(ctx)
		if err != nil {
			return nil, err
		}
		client := historypb.NewHistoryAdminApiClient(conn)
		return apistore.New(client, a.cohortManagerName, src.SourceName()), nil
	case "bolt":
		opts := []boltstore.Option{boltstore.WithLogger(a.logger)}
		if ttl := storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, boltstore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, boltstore.WithMaxCount(ttl.MaxCount))
			}
		}
		return boltstore.NewFromDb(ctx, a.db, src.SourceName(), opts...)
	case "sqlite":
		db, err := a.stores.SqliteHistory(ctx)
		if err != nil {
			return nil, err
		}
		var opts []sqlitestore.WriteOption
		if ttl := storage.TTL; ttl != nil {
			if ttl.MaxAge.Duration > 0 {
				opts = append(opts, sqlitestore.WithMaxAge(ttl.MaxAge.Duration))
			}
			if ttl.MaxCount > 0 {
				opts = append(opts, sqlitestore.WithMaxCount(ttl.MaxCount))
			}
		}
		return db.OpenStore(src.SourceName(), opts...), nil
	default:
		return nil, fmt.Errorf("unsupported storage type %s", storage.Type)
	}
}

func (a *automation) createCollector(store history.Store, traitName trait.Name) (wrap.ServiceUnwrapper, collector, error) {
	switch traitName {
	case allocationpb.TraitName:
		return allocationpb.WrapHistory(historypb.NewAllocationServer(store)), a.collectAllocationChanges, nil
	case trait.AirQualitySensor:
		return airqualitysensorpb.WrapHistory(historypb.NewAirQualitySensorServer(store)), a.collectAirQualityChanges, nil
	case trait.AirTemperature:
		return airtemperaturepb.WrapHistory(historypb.NewAirTemperatureServer(store)), a.collectAirTemperatureChanges, nil
	case trait.Electric:
		return electricpb.WrapHistory(historypb.NewElectricServer(store)), a.collectElectricDemandChanges, nil
	case trait.EnterLeaveSensor:
		return enterleavesensorpb.WrapHistory(historypb.NewEnterLeaveSensorServer(store)), a.collectEnterLeaveEventChanges, nil
	case meterpb.TraitName:
		return meterpb.WrapHistory(historypb.NewMeterServer(store)), a.collectMeterReadingChanges, nil
	case trait.OccupancySensor:
		return occupancysensorpb.WrapHistory(historypb.NewOccupancySensorServer(store)), a.collectOccupancyChanges, nil
	case resourceusepb.TraitName:
		return resourceusepb.WrapHistory(historypb.NewResourceUseServer(store)), a.collectResourceUseChanges, nil
	case soundsensorpb.TraitName:
		return soundsensorpb.WrapHistory(historypb.NewSoundSensorServer(store)), a.collectSoundSensorChanges, nil
	case statuspb.TraitName:
		return statuspb.WrapHistory(historypb.NewStatusServer(store)), a.collectCurrentStatusChanges, nil
	case transportpb.TraitName:
		return transportpb.WrapHistory(historypb.NewTransportServer(store)), a.collectTransportChanges, nil
	default:
		return nil, nil, fmt.Errorf("unsupported trait %s", traitName)
	}
}
