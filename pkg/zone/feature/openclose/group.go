package openclose

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
)

type Group struct {
	traits.UnimplementedOpenCloseApiServer
	apiClient traits.OpenCloseApiClient
	names     []string
	readOnly  bool

	logger *zap.Logger
}

func (g *Group) GetPositions(ctx context.Context, request *traits.GetOpenClosePositionsRequest) (*traits.OpenClosePositions, error) {
	allRes := make([]value, len(g.names))
	fns := make([]func(), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*traits.GetOpenClosePositionsRequest)
		request.Name = name
		i := i
		fns[i] = func() {
			res, err := g.apiClient.GetPositions(ctx, request)
			allRes[i] = value{name: name, val: res, err: err}
		}
	}
	if err := run.InParallel(ctx, run.DefaultConcurrency, fns...); err != nil {
		return nil, err
	}
	return mergeOpenClosePositions(allRes)
}

func (g *Group) UpdatePositions(ctx context.Context, request *traits.UpdateOpenClosePositionsRequest) (*traits.OpenClosePositions, error) {
	if g.readOnly {
		return nil, status.Error(codes.FailedPrecondition, "read-only")
	}
	fns := make([]func() (*traits.OpenClosePositions, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*traits.UpdateOpenClosePositionsRequest)
		request.Name = name
		fns[i] = func() (*traits.OpenClosePositions, error) {
			return g.apiClient.UpdatePositions(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)
	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some openclose.update failed", zap.Errors("errors", multierr.Errors(err)))
		}
	}

	allVals := make([]value, len(g.names))
	for i := range len(fns) {
		// errors are handled above, nil values are handled by the merge
		allVals[i] = value{name: g.names[i], val: allRes[i]}
	}
	return mergeOpenClosePositions(allVals)
}

func (g *Group) Stop(ctx context.Context, request *traits.StopOpenCloseRequest) (*traits.OpenClosePositions, error) {
	if g.readOnly {
		return nil, status.Error(codes.FailedPrecondition, "read-only")
	}
	fns := make([]func() (*traits.OpenClosePositions, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*traits.StopOpenCloseRequest)
		request.Name = name
		fns[i] = func() (*traits.OpenClosePositions, error) {
			return g.apiClient.Stop(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)
	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some openclose.stop failed", zap.Errors("errors", multierr.Errors(err)))
		}
	}

	allVals := make([]value, len(g.names))
	for i := range len(fns) {
		// errors are handled above, nil values are handled by the merge
		allVals[i] = value{name: g.names[i], val: allRes[i]}
	}
	return mergeOpenClosePositions(allVals)
}

func (g *Group) PullPositions(request *traits.PullOpenClosePositionsRequest, server traits.OpenCloseApi_PullPositionsServer) error {
	if len(g.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no openclose names")
	}

	changes := make(chan value)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range g.names {
		request := proto.Clone(request).(*traits.PullOpenClosePositionsRequest)
		request.Name = name
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- value) error {
					stream, err := g.apiClient.PullPositions(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- value{name: request.Name, val: change.OpenClosePosition}
						}
					}
				},
				func(ctx context.Context, changes chan<- value) error {
					res, err := g.apiClient.GetPositions(ctx, &traits.GetOpenClosePositionsRequest{Name: name, ReadMask: request.ReadMask})
					if err != nil {
						return err
					}
					changes <- value{name: request.Name, val: res}
					return nil
				}),
				changes,
			)
		})
	}

	group.Go(func() error {
		// indexes reports which index in values each name has
		indexes := make(map[string]int, len(g.names))
		for i, name := range g.names {
			indexes[name] = i
		}
		values := make([]value, len(g.names))

		var last *traits.OpenClosePositions
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change
				b, err := mergeOpenClosePositions(values)
				if err != nil {
					return err
				}
				filter.Filter(b)

				// don't send duplicates
				if eq(last, b) {
					continue
				}
				last = b

				err = server.Send(&traits.PullOpenClosePositionsResponse{Changes: []*traits.PullOpenClosePositionsResponse_Change{{
					Name:              request.Name,
					ChangeTime:        timestamppb.Now(),
					OpenClosePosition: b,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

type value struct {
	name string
	val  *traits.OpenClosePositions
	err  error
}

func mergeOpenClosePositions(all []value) (*traits.OpenClosePositions, error) {
	switch len(all) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no open close names")
	default:
		// check for errors
		for _, v := range all {
			if v.err != nil {
				return nil, v.err
			}
		}

		positionsByDirection := make(map[traits.OpenClosePosition_Direction][]*traits.OpenClosePosition)
		for _, v := range all {
			if v.val == nil {
				continue
			}
			for _, state := range v.val.States {
				positionsByDirection[state.Direction] = append(positionsByDirection[state.Direction], state)
			}
		}

		out := &traits.OpenClosePositions{}
		for _, pos := range positionsByDirection {
			out.States = append(out.States, mergeOpenClosePosition(pos))
		}
		// sort by direction, ascending
		slices.SortFunc(out.States, func(a, b *traits.OpenClosePosition) int {
			return int(a.Direction) - int(b.Direction)
		})

		// presets are set only if all present presets match by name
	loop:
		for _, v := range all {
			p := v.val.GetPreset()
			switch {
			case out.Preset == nil:
				out.Preset = p // first come, first win
			case p == nil: // skip unset inputs
			case p.Name != out.Preset.Name: // different presets means we don't know, so clear the preset
				out.Preset = nil
				break loop
			}
		}

		return out, nil
	}
}

// mergeOpenClosePosition merges multiple open close positions into a single one.
// All positions must have the same direction.
func mergeOpenClosePosition(all []*traits.OpenClosePosition) *traits.OpenClosePosition {
	out := &traits.OpenClosePosition{}

	presentCount := 0
	for _, pos := range all {
		if pos == nil {
			continue
		}
		presentCount++
		out.Direction = pos.Direction      // these should all be the same
		out.OpenPercent += pos.OpenPercent // will divide later to get average
		// priority order: UNSPECIFIED < SLOW < REDUCED_MOTION < HELD
		// NO_RESISTANCE would go here ^
		switch pos.Resistance {
		case traits.OpenClosePosition_RESISTANCE_UNSPECIFIED: // keep existing
		case traits.OpenClosePosition_SLOW:
			switch out.Resistance {
			case traits.OpenClosePosition_REDUCED_MOTION, traits.OpenClosePosition_HELD: // keep existing
			default:
				out.Resistance = pos.Resistance
			}
		case traits.OpenClosePosition_REDUCED_MOTION:
			switch out.Resistance {
			case traits.OpenClosePosition_HELD: // keep existing
			default:
				out.Resistance = pos.Resistance
			}
		case traits.OpenClosePosition_HELD:
			out.Resistance = pos.Resistance
		}
	}
	if presentCount == 0 {
		return out
	}
	out.OpenPercent /= float32(presentCount)

	// NB: no tween support here

	return out
}
