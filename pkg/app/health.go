package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/internal/health/healthdb"
	"github.com/smart-core-os/sc-bos/internal/health/healthhistory"
	"github.com/smart-core-os/sc-bos/pkg/app/files"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

// setupHealthRegistry returns a healthpb.Registry that is integrated with the deviceStore and announced on the rootNode.
//
// The returned localChecks mirrors the registry: one device per name holding only the checks this node evaluates,
// measured values included.
// Unlike deviceStore it never holds checks written by other means, such as a gateway re-announcing its cohort's checks.
func setupHealthRegistry(ctx context.Context, config sysconf.Config, deviceStore *devicespb.Collection, rootNode node.Announcer, logger *zap.Logger) (_ *healthpb.Registry, localChecks *devicespb.Collection, close func() error, _ error) {
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
	healthCheckStore, err := healthdb.Open(ctx, files.Path(config.DataDir, sysconf.HealthDBPath), dbOpts...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("health check store: %w", err)
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
	localChecks = devicespb.NewCollection()

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
				node.HasServer[healthpb.HealthApiServer](healthpb.RegisterHealthApiServer, healthpb.NewModelServer(m)),
				node.HasServer[healthpb.HealthHistoryServer](healthpb.RegisterHealthHistoryServer, healthHistoryServer),
			)
			announcedChecks[name] = checkedDevice{undo: undo, m: m}
		}),
		healthpb.WithOnCheckCreate(func(name string, c *healthpb.HealthCheck) *healthpb.HealthCheck {
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
			} else if _, err := existing.m.CreateHealthCheck(c); err != nil {
				logger.Error("seed health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}

			// update the devices api
			_, err := deviceStore.Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
				dstDev := dst.(*devicespb.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, removeMeasuredValues(c))
			}))
			if err != nil {
				logger.Error("update device with health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}

			// update the local checks mirror
			if err := setLocalCheck(localChecks, name, c, resource.WithCreateIfAbsent()); err != nil {
				logger.Error("update local checks with health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}
			return c
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *healthpb.HealthCheck) {
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
				// see SCB-1508, carry on so the stores below still track the registry
				logger.Error("update health check for unknown name", zap.String("name", name), zap.String("checkId", c.Id))
			} else if _, err := a.m.UpdateHealthCheck(c); err != nil {
				logger.Error("update health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}

			// update the devices api
			_, err = deviceStore.Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, _ proto.Message) {
				dstDev := dst.(*devicespb.Device)
				dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, removeMeasuredValues(c))
			}))
			if err != nil {
				logger.Error("update device with health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}

			// update the local checks mirror
			if err := setLocalCheck(localChecks, name, c); err != nil {
				logger.Error("update local checks with health check", zap.String("name", name), zap.String("checkId", c.Id), zap.Error(err))
			}
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			// update the health api
			announcedChecksMu.Lock()
			defer announcedChecksMu.Unlock()
			a, ok := announcedChecks[name]
			if !ok {
				logger.Error("delete health check for unknown name", zap.String("name", name), zap.String("checkId", id))
			} else if err := a.m.DeleteHealthCheck(id); err != nil {
				logger.Error("delete health check", zap.String("name", name), zap.String("checkId", id), zap.Error(err))
			}

			_, err := deviceStore.Update(&devicespb.Device{Name: name}, resource.WithMerger(func(_ *masks.FieldUpdater, dst, _ proto.Message) {
				dstDev := dst.(*devicespb.Device)
				dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
			}))
			if err != nil {
				logger.Error("update device removing health check", zap.String("name", name), zap.String("checkId", id), zap.Error(err))
			}

			// update the local checks mirror
			_, err = localChecks.Update(&devicespb.Device{Name: name}, resource.WithMerger(func(_ *masks.FieldUpdater, dst, _ proto.Message) {
				dstDev := dst.(*devicespb.Device)
				dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
			}))
			if err != nil && status.Code(err) != codes.NotFound {
				logger.Error("update local checks removing health check", zap.String("name", name), zap.String("checkId", id), zap.Error(err))
			}
		}),
		healthpb.WithOnNameDelete(func(name string) {
			// A check can be created for name between the registry forgetting it and this callback (SCB-1508),
			// so only remove the local checks mirror entry if it is still empty.
			_, err := localChecks.Delete(name, resource.WithAllowMissing(true), resource.WithExpectedCheck(func(msg proto.Message) error {
				if len(msg.(*devicespb.Device).GetHealthChecks()) > 0 {
					return errLocalChecksNotEmpty
				}
				return nil
			}))
			if err != nil && !errors.Is(err, errLocalChecksNotEmpty) {
				logger.Error("delete local checks", zap.String("name", name), zap.Error(err))
			}

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
	return checkRegistry, localChecks, close, nil
}

var errLocalChecksNotEmpty = errors.New("local checks not empty")

// setLocalCheck records a copy of c against name in localChecks, replacing any check with the same id.
// A NotFound error, for a name localChecks has no entry for, is ignored.
func setLocalCheck(localChecks *devicespb.Collection, name string, c *healthpb.HealthCheck, opts ...resource.WriteOption) error {
	c = proto.Clone(c).(*healthpb.HealthCheck)
	opts = append([]resource.WriteOption{resource.WithMerger(func(_ *masks.FieldUpdater, dst, _ proto.Message) {
		dstDev := dst.(*devicespb.Device)
		dstDev.Name = name
		dstDev.HealthChecks = healthpb.SetCheck(dstDev.HealthChecks, c)
	})}, opts...)
	_, err := localChecks.Update(&devicespb.Device{Name: name}, opts...)
	if status.Code(err) == codes.NotFound {
		return nil
	}
	return err
}

// removeMeasuredValues returns a copy of c with any measured values removed.
func removeMeasuredValues(c *healthpb.HealthCheck) *healthpb.HealthCheck {
	// We do the removal here, instead of in the DevicesApi server,
	// because here is the write point for those devices.
	// Doing the removal on write means we don't have to worry about
	// filtering the properties during reads, queries, or any of that.

	if c == nil {
		return nil
	}
	// Always clone: c is typically the registry's own copy, and the result may be stored in the device store as is.
	c = proto.Clone(c).(*healthpb.HealthCheck)
	if c.GetBounds().GetCurrentValue() != nil {
		c.GetBounds().CurrentValue = nil
	}
	return c
}
