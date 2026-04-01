package lighting

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/merge"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

// Group implements traits.LightApiServer backed by a group of lights.
type Group struct {
	lightpb.UnimplementedLightApiServer
	lightpb.UnimplementedLightInfoServer
	client   lightpb.LightApiClient
	info     lightpb.LightInfoClient
	names    []string
	readOnly bool

	logger *zap.Logger
}

func (g *Group) UpdateBrightness(ctx context.Context, request *lightpb.UpdateBrightnessRequest) (*lightpb.Brightness, error) {
	if g.readOnly {
		return nil, status.Error(codes.FailedPrecondition, "read-only")
	}
	fns := make([]func() (*lightpb.Brightness, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*lightpb.UpdateBrightnessRequest)
		request.Name = name
		fns[i] = func() (*lightpb.Brightness, error) {
			return g.client.UpdateBrightness(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some lights failed", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeBrightness(allRes)
}

func (g *Group) GetBrightness(ctx context.Context, request *lightpb.GetBrightnessRequest) (*lightpb.Brightness, error) {
	fns := make([]func() (*lightpb.Brightness, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*lightpb.GetBrightnessRequest)
		request.Name = name
		fns[i] = func() (*lightpb.Brightness, error) {
			return g.client.GetBrightness(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some lights failed", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeBrightness(allRes)
}

func (g *Group) PullBrightness(request *lightpb.PullBrightnessRequest, server lightpb.LightApi_PullBrightnessServer) error {
	if len(g.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no light names")
	}

	type c struct {
		name string
		val  *lightpb.Brightness
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range g.names {
		request := proto.Clone(request).(*lightpb.PullBrightnessRequest)
		request.Name = name
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullBrightness(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{name: request.Name, val: change.Brightness}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetBrightness(ctx, &lightpb.GetBrightnessRequest{Name: name, ReadMask: request.ReadMask})
					if err != nil {
						return err
					}
					changes <- c{name: request.Name, val: res}
					return nil
				}),
				changes,
			)
		})
	}

	group.Go(func() error {
		// indexes reports which index in values each name name has
		indexes := make(map[string]int, len(g.names))
		for i, name := range g.names {
			indexes[name] = i
		}
		values := make([]*lightpb.Brightness, len(g.names))

		var last *lightpb.Brightness
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change.val
				b, err := mergeBrightness(values)
				if err != nil {
					return err
				}
				filter.Filter(b)

				// don't send duplicates
				if eq(last, b) {
					continue
				}
				last = b

				err = server.Send(&lightpb.PullBrightnessResponse{Changes: []*lightpb.PullBrightnessResponse_Change{{
					Name:       request.Name,
					ChangeTime: timestamppb.Now(),
					Brightness: b,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func mergeBrightness(allRes []*lightpb.Brightness) (*lightpb.Brightness, error) {
	switch len(allRes) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no light names")
	case 1:
		return allRes[0], nil
	default:
		out := &lightpb.Brightness{}
		var l float32
		for _, b := range allRes {
			if b != nil {
				proto.Merge(out, b)
				l++
			}
		}
		var averageBrightness float32
		for _, b := range allRes {
			if b != nil {
				averageBrightness += b.LevelPercent / l
			}
		}
		out.LevelPercent = averageBrightness
		return out, nil
	}
}

func (g *Group) DescribeBrightness(ctx context.Context, request *lightpb.DescribeBrightnessRequest) (*lightpb.BrightnessSupport, error) {
	if g.info == nil {
		return nil, status.Error(codes.Unimplemented, "info not supported")
	}
	fns := make([]func() (*lightpb.BrightnessSupport, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*lightpb.DescribeBrightnessRequest)
		request.Name = name
		fns[i] = func() (*lightpb.BrightnessSupport, error) {
			return g.info.DescribeBrightness(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}
	// err ignored here — some lights may not support the info aspect

	desc, err := mergeDescription(allRes)
	if err != nil {
		return nil, err
	}
	if g.readOnly {
		desc.ResourceSupport.Writable = false
	}
	return desc, err
}

func mergeDescription(allRes []*lightpb.BrightnessSupport) (*lightpb.BrightnessSupport, error) {
	switch len(allRes) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no light names")
	case 1:
		return allRes[0], nil
	default:
		out := &lightpb.BrightnessSupport{}
		out.ResourceSupport = merge.ResourceSupport(allRes, func(s *lightpb.BrightnessSupport) *typespb.ResourceSupport {
			return s.GetResourceSupport()
		})
		out.BrightnessAttributes = merge.Int32Attributes(allRes, func(s *lightpb.BrightnessSupport) *typespb.Int32Attributes {
			return s.GetBrightnessAttributes()
		})

		// Find a unique set of the presets from all the lights
		// We want to preserve as much order as we can in case it's important,
		// so instead of using a sorted slice approach we use a map to track duplicates.
		seenPresets := make(map[string]struct{})
		for _, item := range allRes {
			for _, preset := range item.GetPresets() {
				if _, ok := seenPresets[preset.Name]; ok {
					continue
				}
				seenPresets[preset.Name] = struct{}{}
				out.Presets = append(out.Presets, preset)
			}
		}
		return out, nil
	}
}
