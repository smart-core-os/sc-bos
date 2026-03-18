package mode

import (
	"context"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/modepb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/mode/config"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

// MixedValue is used as ModeValues.Value when underlying devices disagree on the actual value for a mode.
const MixedValue = "<< mixed >>"

type Group struct {
	modepb.UnimplementedModeApiServer
	modepb.UnimplementedModeInfoServer

	client modepb.ModeApiClient
	cfg    config.Root

	logger *zap.Logger
}

func (g *Group) DescribeModes(_ context.Context, _ *modepb.DescribeModesRequest) (*modepb.ModesSupport, error) {
	dst := &modepb.ModesSupport{AvailableModes: &modepb.Modes{}}
	for mode, options := range g.cfg.Modes {
		modeMsg := &modepb.Modes_Mode{
			Name:    mode,
			Ordered: false,
		}
		for _, option := range options {
			modeMsg.Values = append(modeMsg.Values, &modepb.Modes_Value{
				Name: option.Name,
			})
		}
		dst.AvailableModes.Modes = append(dst.AvailableModes.Modes, modeMsg)
	}
	return dst, nil
}

func (g *Group) GetModeValues(ctx context.Context, request *modepb.GetModeValuesRequest) (*modepb.ModeValues, error) {
	names := g.cfg.AllDeviceNames()
	if len(names) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "zone has no mode names")
	}
	fns := make([]func() (*modepb.ModeValues, error), len(names))
	for i, name := range names {
		name := name
		fns[i] = func() (*modepb.ModeValues, error) {
			return g.client.GetModeValues(ctx, &modepb.GetModeValuesRequest{Name: name})
		}
	}

	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some modes failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}

	all := make(map[string]*modepb.ModeValues)
	for i, res := range allRes {
		if res == nil {
			continue
		}
		all[names[i]] = res
	}

	return g.mergeModeValues(all), nil
}

func (g *Group) UpdateModeValues(ctx context.Context, request *modepb.UpdateModeValuesRequest) (*modepb.ModeValues, error) {
	values := request.ModeValues
	// remove any values that we've been sent that shouldn't be written
	masks.NewResponseFilter(masks.WithFieldMask(request.UpdateMask)).Filter(values)
	all := g.unmergeModeValues(values)
	type r struct {
		name string
		val  *modepb.ModeValues
		err  error
	}
	results := make([]r, len(all))
	i := 0
	for name, modeValues := range all {
		results[i] = r{name: name, val: modeValues}
		i++
	}

	var wg sync.WaitGroup
	wg.Add(len(results))
	for i, result := range results {
		i := i
		result := result
		go func() {
			defer wg.Done()
			var updateMask *fieldmaskpb.FieldMask
			// todo: this currently doesn't work because fieldbaskpb.FieldMask.IsValid fails with map keys!
			//  if/when it does or we write our own validation logic in sc-bos we can't use field masks :(
			// updateMask = &fieldmaskpb.FieldMask{}
			// for k := range result.val.Values {
			// 	updateMask.Paths = append(updateMask.Paths, fmt.Sprintf("values.%s", k))
			// }
			val, err := g.client.UpdateModeValues(ctx, &modepb.UpdateModeValuesRequest{
				Name:       result.name,
				ModeValues: result.val,
				UpdateMask: updateMask,
			})
			results[i].val = val
			results[i].err = err
		}()
	}

	wgDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgDone)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-wgDone:
	}

	var allErrs []error
	all = make(map[string]*modepb.ModeValues)
	for _, r := range results {
		if r.err != nil {
			allErrs = append(allErrs, r.err)
			continue
		}
		all[r.name] = r.val
	}

	if len(allErrs) == len(results) {
		return nil, multierr.Combine(allErrs...)
	}
	if len(allErrs) > 0 {
		if g.logger != nil {
			g.logger.Warn("some modes failed to update",
				zap.Int("success", len(results)-len(allErrs)), zap.Int("failed", len(allErrs)),
				zap.Errors("errors", allErrs))
		}
	}
	return g.mergeModeValues(all), nil
}

func (g *Group) PullModeValues(request *modepb.PullModeValuesRequest, server modepb.ModeApi_PullModeValuesServer) error {
	names := g.cfg.AllDeviceNames()
	if len(names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no mode names")
	}

	type c struct {
		name string
		val  *modepb.ModeValues
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range names {
		name := name
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullModeValues(ctx, &modepb.PullModeValuesRequest{Name: name})
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							err := chans.SendContext(ctx, changes, c{name: name, val: change.ModeValues})
							if err != nil {
								return err
							}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetModeValues(ctx, &modepb.GetModeValuesRequest{Name: name})
					if err != nil {
						return err
					}
					return chans.SendContext(ctx, changes, c{name: name, val: res})
				},
			), changes)
		})
	}

	group.Go(func() error {
		all := make(map[string]*modepb.ModeValues, len(names))

		var last *modepb.ModeValues
		eq := cmp.Equal()
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				all[change.name] = change.val
				values := g.mergeModeValues(all)
				filter.Filter(values)
				if eq(last, values) {
					continue
				}
				last = values
				err := server.Send(&modepb.PullModeValuesResponse{Changes: []*modepb.PullModeValuesResponse_Change{
					{
						Name:       request.Name,
						ChangeTime: timestamppb.Now(),
						ModeValues: values,
					},
				}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func (g *Group) mergeModeValues(all map[string]*modepb.ModeValues) *modepb.ModeValues {
	type dstModeValue struct {
		name, mode, value string
	}
	var dstModeValues []dstModeValue
	for dstMode, options := range g.cfg.Modes {
		for _, option := range options {
			for _, source := range option.Sources {
				for _, device := range source.Devices {
					srcValues, ok := all[device]
					if !ok {
						continue
					}

					for srcMode, srcValue := range srcValues.Values {
						if srcMode != source.Mode || srcValue != source.Value {
							continue
						}

						dstModeValues = append(dstModeValues, dstModeValue{
							name:  device,
							mode:  dstMode,
							value: option.Name,
						})
					}
				}
			}
		}
	}
	dst := &modepb.ModeValues{Values: make(map[string]string)}
	for _, value := range dstModeValues {
		if old, ok := dst.Values[value.mode]; ok {
			if old != value.value {
				dst.Values[value.mode] = MixedValue
			}
			continue
		}
		dst.Values[value.mode] = value.value
	}
	return dst
}

func (g *Group) unmergeModeValues(values *modepb.ModeValues) map[string]*modepb.ModeValues {
	all := make(map[string]*modepb.ModeValues)
	for srcMode, srcValue := range values.Values {
		options, ok := g.cfg.Modes[srcMode]
		if !ok {
			continue
		}
		for _, option := range options {
			if option.Name != srcValue {
				continue
			}
			for _, source := range option.Sources {
				for _, device := range source.Devices {
					if _, ok := all[device]; !ok {
						all[device] = &modepb.ModeValues{Values: make(map[string]string)}
					}
					all[device].Values[source.Mode] = source.Value
				}
			}
		}
	}
	return all
}
