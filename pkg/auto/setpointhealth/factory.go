// Package setpointhealth provides an automation that watches a device's measured value against its
// set point and updates a health check when the measurement fails to track the set point for a
// sustained period of time.
package setpointhealth

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/internal/anytrait"
	"github.com/smart-core-os/sc-bos/pkg/auto/setpointhealth/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

const AutoName = "setpointhealth"

var Factory auto.Factory = factory{}

type factory struct{}

func (f factory) New(services auto.Services) service.Lifecycle {
	a := &impl{
		Services: services,
	}
	a.Logger = a.Logger.Named(AutoName)
	a.Service = service.New[config.Root](service.MonoApply(a.applyConfig), service.WithParser(config.Read))
	return a
}

type impl struct {
	auto.Services
	*service.Service[config.Root]
}

func (a *impl) applyConfig(ctx context.Context, cfg config.Root) error {
	devicesMask, err := fieldmaskpb.New(&devicespb.Device{}, "name")
	if err != nil {
		return err
	}
	devicesApi := a.Devices

	go func() {
		runningChecks := make(map[string]func())
		defer func() {
			for _, stop := range runningChecks {
				stop()
			}
		}()
		// the task is configured to retry forever (until ctx is done) so the error is ignored.
		_ = task.Run(ctx, func(ctx context.Context) (task.Next, error) {
			stream, err := devicesApi.PullDevices(ctx, &devicespb.PullDevicesRequest{
				ReadMask: devicesMask,
				Query:    &devicespb.Device_Query{Conditions: cfg.DevicesPb()},
			})
			if err != nil {
				return task.Normal, err
			}
			for {
				res, err := stream.Recv()
				if err != nil {
					return task.ResetBackoff, err
				}

				for _, change := range res.GetChanges() {
					ov, nv := change.GetOldValue(), change.GetNewValue()
					switch {
					case ov == nil && nv == nil, ov != nil && nv != nil:
						// do nothing, neither added nor removed
					case ov == nil && nv != nil:
						// added
						// sanity check
						if _, ok := runningChecks[change.GetName()]; ok {
							a.Logger.Warn("repeated ADD from PullDevices", zap.String("device", change.GetName()))
							continue
						}
						stop, err := a.newCheck(ctx, nv, cfg)
						if err != nil {
							a.Logger.Error("failed to create health check", zap.String("device", change.GetName()), zap.Error(err))
							continue
						}
						runningChecks[change.GetName()] = stop
					case ov != nil && nv == nil:
						// removed
						if stop, ok := runningChecks[change.GetName()]; ok {
							stop()
							delete(runningChecks, change.GetName())
						}
					}
				}
			}
		}, task.WithRetry(task.RetryUnlimited), task.WithBackoff(100*time.Millisecond, time.Minute))
	}()
	return nil
}

func (a *impl) newCheck(ctx context.Context, device *devicespb.Device, cfg config.Root) (func(), error) {
	source := cfg.Source

	// find the trait resource we are checking
	t, err := anytrait.FindByName(source.Trait)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source.Trait, err)
	}
	var r anytrait.Resource
	if source.Resource == "" {
		resources := t.Resources()
		if len(resources) == 0 {
			return nil, fmt.Errorf("trait %q has no resources", source.Trait)
		}
		r = resources[0]
	} else {
		for _, res := range t.Resources() {
			if source.Resource.String() == res.Name() {
				r = res
				break
			}
		}
		if r.Name() == "" {
			return nil, fmt.Errorf("trait %q has no resource %q", source.Trait, source.Resource)
		}
	}

	// check both value paths are resolvable, and build a combined read mask
	measPath, measMask, err := source.Measured.Parse(r.Message())
	if err != nil {
		return nil, fmt.Errorf("source measured path %q not found in %s[%s]: %w", source.Measured, source.Trait, r.Name(), err)
	}
	if err := numericLeaf(measPath); err != nil {
		return nil, fmt.Errorf("source measured path %q in %s[%s]: %w", source.Measured, source.Trait, r.Name(), err)
	}
	spPath, spMask, err := source.SetPoint.Parse(r.Message())
	if err != nil {
		return nil, fmt.Errorf("source setPoint path %q not found in %s[%s]: %w", source.SetPoint, source.Trait, r.Name(), err)
	}
	if err := numericLeaf(spPath); err != nil {
		return nil, fmt.Errorf("source setPoint path %q in %s[%s]: %w", source.SetPoint, source.Trait, r.Name(), err)
	}
	readMask := unionFieldMasks(measMask, spMask)

	logger := a.Logger.With(
		zap.String("device", device.GetName()),
		zap.Stringer("trait", source.Trait),
		zap.String("resource", r.Name()),
		zap.Stringer("measured", source.Measured),
		zap.Stringer("setPoint", source.SetPoint),
	)

	// make the check instance.
	// clone because NewFaultCheck modifies the config; a missing check still needs an empty message.
	checkCfg := cfg.CheckPb()
	if checkCfg == nil {
		checkCfg = &healthpb.HealthCheck{}
	} else {
		checkCfg = proto.Clone(checkCfg).(*healthpb.HealthCheck)
	}
	check, err := a.Health.NewFaultCheck(device.GetName(), checkCfg)
	if err != nil {
		return nil, fmt.Errorf("create check: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	// set up the value watcher
	changes := make(chan anytrait.Value)
	fetcher := r.Fetcher(a.Node.ClientConn(), anytrait.ReadRequest{
		Name:     device.GetName(),
		ReadMask: readMask,
	})
	g.Go(func() error {
		defer close(changes)
		return pull.Changes(ctx, fetcher, changes, pull.WithLogger(logger))
	})

	// react to value changes
	w := &worker{
		check:       check,
		measured:    measPath,
		setPoint:    spPath,
		tolerance:   cfg.Tolerance,
		duration:    cfg.Duration.Duration,
		maxDuration: cfg.MaxDuration.Duration,
		device:      device.GetName(),
		logger:      logger,
	}
	g.Go(func() error {
		return w.run(ctx, changes)
	})

	return func() {
		cancel()
		check.Dispose()
	}, nil
}

// unionFieldMasks combines the paths of the given field masks into a single mask, dropping nils and
// duplicate paths. A nil result means "read everything".
func unionFieldMasks(masks ...*fieldmaskpb.FieldMask) *fieldmaskpb.FieldMask {
	var paths []string
	seen := make(map[string]struct{})
	for _, m := range masks {
		if m == nil {
			return nil // one path wants everything, so read everything
		}
		for _, p := range m.GetPaths() {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return &fieldmaskpb.FieldMask{Paths: paths}
}
