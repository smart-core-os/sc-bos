// Package healthbounds provides an automation that watches device traits for values that exceed normal bounds and updates health checks based on the results.
package healthbounds

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vanti-dev/sc-bos/internal/protobuf/protopath2"
	"github.com/vanti-dev/sc-bos/pkg/auto"
	"github.com/vanti-dev/sc-bos/pkg/auto/healthbounds/config"
	"github.com/vanti-dev/sc-bos/pkg/auto/healthbounds/internal/anytrait"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb"
	"github.com/vanti-dev/sc-bos/pkg/task"
	"github.com/vanti-dev/sc-bos/pkg/task/service"
	"github.com/vanti-dev/sc-bos/pkg/util/pull"
)

const AutoName = "healthbounds"

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
	devicesMask, err := fieldmaskpb.New(&gen.Device{}, "name")
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
			stream, err := devicesApi.PullDevices(ctx, &gen.PullDevicesRequest{
				ReadMask: devicesMask,
				Query:    &gen.Device_Query{Conditions: cfg.DevicesPb()},
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
						stop, err := a.newCheck(ctx, nv, cfg.CheckPb(), cfg.Source)
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

func (a *impl) newCheck(ctx context.Context, device *gen.Device, checkCfg *gen.HealthCheck, source config.Source) (func(), error) {
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
			if string(source.Resource) == res.Name() {
				r = res
				break
			}
		}
		if r.Name() == "" {
			return nil, fmt.Errorf("trait %q has no resource %q", source.Trait, source.Resource)
		}
	}

	logger := a.Logger.With(
		zap.String("device", device.GetName()),
		zap.Stringer("trait", source.Trait),
		zap.String("resource", r.Name()),
		zap.Stringer("value", source.Value),
	)

	// check the value is resolvable
	rpath, err := protopath2.ParsePath(r.Message(), source.Value.ToProto())
	if err != nil {
		return nil, fmt.Errorf("source value path %q not found in %s[%s]: %w", source.Value, source.Trait, r.Name(), err)
	}

	// make the check instance
	checkCfg = proto.Clone(checkCfg).(*gen.HealthCheck) // clone because NewBoundsCheck modifies the config
	check, err := a.Health.NewBoundsCheck(device.GetName(), checkCfg)
	if err != nil {
		return nil, fmt.Errorf("create check: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	// set up the value watcher
	changes := make(chan anytrait.Value)
	fetcher := resourceFetcher(a.Node.ClientConn(), anytrait.ReadRequest{
		Name:     device.Name,
		ReadMask: source.Value.ToFieldMask(),
	}, r)
	g.Go(func() error {
		defer close(changes)
		return pull.Changes(ctx, fetcher, changes, pull.WithLogger(a.Logger.With(zap.String("device", device.Name))))
	})

	// react to value changes
	g.Go(func() error {
		for change := range changes {
			values, err := protopath2.PathValues(rpath, change.Proto())
			if err != nil {
				logger.Debug("value path extraction failed", zap.Error(err))
				err := fmt.Errorf("failed to read %s.%s[%q] from %q: %w", source.Trait, r.Name(), source.Value, device.GetName(), err)
				check.UpdateReliability(ctx, healthpb.ReliabilityFromErr(err))
				continue
			}
			healthVal, err := healthValueFromReflectValue(values)
			if err != nil {
				logger.Debug("health value conversion failed", zap.Any("path", values), zap.Error(err))
				err := fmt.Errorf("failed to convert %s.%s[%q] from %q to health value: %w", source.Trait, r.Name(), source.Value, device.GetName(), err)
				check.UpdateReliability(ctx, healthpb.ReliabilityFromErr(err))
				continue
			}
			check.UpdateValue(ctx, healthVal)
		}
		return nil
	})
	return func() {
		cancel()
		check.Dispose()
	}, nil
}

func resourceFetcher(conn grpc.ClientConnInterface, req anytrait.ReadRequest, r anytrait.Resource) pull.Fetcher[anytrait.Value] {
	return pull.NewFetcher(
		func(ctx context.Context, changes chan<- anytrait.Value) error {
			stream, err := r.Pull(ctx, conn, anytrait.PullRequest{
				ReadRequest: req,
			})
			if err != nil {
				return err
			}
			for {
				res, err := stream.Recv()
				if err != nil {
					return err
				}
				for _, change := range res.Changes {
					changes <- change.Value
				}
			}
		},
		func(ctx context.Context, changes chan<- anytrait.Value) error {
			value, err := r.Get(ctx, conn, anytrait.GetRequest{
				ReadRequest: req,
			})
			if err != nil {
				return err
			}
			changes <- value
			return nil
		},
	)
}

func healthValueFromReflectValue(path protopath.Values) (*gen.HealthCheck_Value, error) {
	lastStep := path.Index(-1)
	switch goValue := lastStep.Value.Interface().(type) {
	case nil:
		return nil, nil
	case bool:
		return healthpb.BoolValue(goValue), nil
	case int32:
		return healthpb.IntValue(int64(goValue)), nil
	case int64:
		return healthpb.IntValue(goValue), nil
	case uint32:
		return healthpb.UintValue(uint64(goValue)), nil
	case uint64:
		return healthpb.UintValue(goValue), nil
	case float32:
		return healthpb.FloatValue(float64(goValue)), nil
	case float64:
		return healthpb.FloatValue(goValue), nil
	case string:
		return healthpb.StringValue(goValue), nil
	case *timestamppb.Timestamp:
		return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_TimestampValue{TimestampValue: goValue}}, nil
	case *durationpb.Duration:
		return &gen.HealthCheck_Value{Value: &gen.HealthCheck_Value_DurationValue{DurationValue: goValue}}, nil
	case protoreflect.EnumNumber:
		// use the enum name instead of its number where we can
		fd := lastFieldDescriptor(path)
		if fd == nil || fd.Kind() != protoreflect.EnumKind {
			return healthpb.IntValue(int64(goValue)), nil
		}
		return healthpb.StringValue(string(fd.Enum().Values().ByNumber(goValue).Name())), nil
	default:
		return nil, fmt.Errorf("unsupported value type: %T", goValue)
	}
}

// lastFieldDescriptor returns a FieldDescriptor describing the value at the end of the path.
// When the path ends in a map index step, the descriptor for the map value is returned.
// When the path ends in a list index step, the descriptor for the list is returned.
func lastFieldDescriptor(path protopath.Values) protoreflect.FieldDescriptor {
	if len(path.Path) == 0 {
		return nil
	}
	lastStep := path.Index(-1)
	switch lastStep.Step.Kind() {
	case protopath.FieldAccessStep:
		return lastStep.Step.FieldDescriptor()
	case protopath.MapIndexStep:
		fieldStep := path.Index(-2)
		return fieldStep.Step.FieldDescriptor().MapValue()
	case protopath.ListIndexStep:
		fieldStep := path.Index(-2)
		return fieldStep.Step.FieldDescriptor()
	default:
		return nil
	}
}
