package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/vanti-dev/sc-bos/internal/health/healthdb"
	"github.com/vanti-dev/sc-bos/internal/health/healthhistory"
	"github.com/vanti-dev/sc-bos/pkg/app/files"
	"github.com/vanti-dev/sc-bos/pkg/app/sysconf"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/devicespb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb"
	"github.com/vanti-dev/sc-bos/pkg/node"
)

// setupHealthRegistry returns a healthpb.Registry that is integrated with the deviceStore and announced on the rootNode.
func setupHealthRegistry(ctx context.Context, config sysconf.Config, deviceStore *devicespb.Collection, rootNode node.Announcer, logger *zap.Logger) (_ *healthpb.Registry, close func() error, _ error) {
	// persistent storage for health checks and history
	var dbOpts []healthdb.Option
	if config.Health.TTL.MaxCount != nil || config.Health.TTL.MaxAge != nil {
		// note: a min-count means nothing on its own
		var minCount, maxCount int64
		var maxAge time.Duration
		if v := config.Health.TTL.MinCount; v != nil {
			minCount = int64(*v)
		}
		if v := config.Health.TTL.MaxCount; v != nil {
			maxCount = int64(*v)
		}
		if v := config.Health.TTL.MaxAge; v != nil {
			maxAge = v.Duration
		}
		dbOpts = append(dbOpts, healthdb.WithTrimOnWrite(minCount, maxCount, maxAge))
	}
	healthCheckStore, err := healthdb.Open(ctx, files.Path(config.DataDir, config.Health.DBPath), dbOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("health check store: %w", err)
	}
	close = healthCheckStore.Close
	// history (including seeding) support
	checkSeeder := healthhistory.NewSeeder(healthCheckStore)
	checkRecorder := healthhistory.NewRecorder(healthCheckStore)
	healthHistoryServer := healthhistory.NewServer(healthCheckStore)
	// History api registration and device metadata/parent support
	type checkedDevice struct {
		undo node.Undo
		m    *healthpb.Model
	}
	var announcedChecksMu sync.Mutex
	announcedChecks := make(map[string]checkedDevice)

	checkRegistry := healthpb.NewRegistry(
		healthpb.WithOnNameCreate(func(name string) {
			// announce that the name implements the health trait
			announcedChecksMu.Lock()
			defer announcedChecksMu.Unlock()
			if _, ok := announcedChecks[name]; ok {
				logger.Error("health check already exists for name", zap.String("name", name))
				return
			}
			m := healthpb.NewModel()
			undo := rootNode.Announce(name,
				node.HasTrait(healthpb.TraitName),
				node.HasServer[gen.HealthApiServer](gen.RegisterHealthApiServer, healthpb.NewModelServer(m)),
				node.HasServer[gen.HealthHistoryServer](gen.RegisterHealthHistoryServer, healthHistoryServer),
			)
			announcedChecks[name] = checkedDevice{undo: undo, m: m}
		}),
		healthpb.WithOnCheckCreate(func(name string, c *gen.HealthCheck) *gen.HealthCheck {
			// seed from history if we can
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			oldCheck := checkSeeder.Seed(ctx, name, c)
			if oldCheck != nil {
				c = oldCheck
			}

			// update the health api
			announcedChecksMu.Lock()
			defer announcedChecksMu.Unlock()
			// seed the model
			existing, ok := announcedChecks[name]
			if !ok {
				logger.Error("create health check for unknown name", zap.String("name", name), zap.String("checkId", c.Id))
			}
			_, err := existing.m.CreateHealthCheck(c)
			if err != nil {
				logger.Error("seed health check", zap.String("name", name), zap.String("checkId", oldCheck.Id), zap.Error(err))
			}

			// update the devices api
			_, err = deviceStore.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}))
			if err != nil {
				logger.Error("update device with health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}
			return c
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *gen.HealthCheck) {
			// save the update to history
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := checkRecorder.Record(ctx, name, c)
			if err != nil {
				logger.Error("record health check update", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}

			// update the health api
			announcedChecksMu.Lock()
			defer announcedChecksMu.Unlock()
			a, ok := announcedChecks[name]
			if !ok {
				logger.Error("update health check for unknown name", zap.String("name", name), zap.String("checkId", c.Id))
				return
			}
			_, err = a.m.UpdateHealthCheck(c)
			if err != nil {
				logger.Error("update health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}

			// update the devices api
			_, err = deviceStore.Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, _ proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, c)
			}))
			if err != nil {
				logger.Error("update device with health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			// update the health api
			announcedChecksMu.Lock()
			defer announcedChecksMu.Unlock()
			a, ok := announcedChecks[name]
			if !ok {
				logger.Error("delete health check for unknown name", zap.String("name", name), zap.String("checkId", id))
			}
			err := a.m.DeleteHealthCheck(id)
			if err != nil {
				logger.Error("delete health check", zap.String("name", name), zap.String("checkId", id), zap.Error(err))
			}

			_, err = deviceStore.Update(&gen.Device{Name: name}, resource.WithMerger(func(_ *masks.FieldUpdater, dst, _ proto.Message) {
				dstDev := dst.(*gen.Device)
				dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
			}))
			if err != nil {
				logger.Error("update device removing health check", zap.String("name", name), zap.String("checkId", id), zap.Error(err))
			}
		}),
		healthpb.WithOnNameDelete(func(name string) {
			// unannounce the health trait
			announcedChecksMu.Lock()
			defer announcedChecksMu.Unlock()
			a, ok := announcedChecks[name]
			if !ok {
				logger.Error("unannounce health check for unknown name", zap.String("name", name), zap.String("checkId", name))
				return
			}
			a.undo()
			delete(announcedChecks, name)

			// note: the deviceStore doesn't need updating because undoing will manage that
		}),
	)
	return checkRegistry, close, nil
}
