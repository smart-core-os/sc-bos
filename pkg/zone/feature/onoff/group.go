package onoff

import (
	"context"
	"iter"
	"maps"
	"slices"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

type Group struct {
	onoffpb.UnimplementedOnOffApiServer
	client   onoffpb.OnOffApiClient
	names    []string
	readOnly bool

	logger *zap.Logger
}

func (g *Group) GetOnOff(ctx context.Context, request *onoffpb.GetOnOffRequest) (*onoffpb.OnOff, error) {
	fns := make([]func() (*onoffpb.OnOff, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*onoffpb.GetOnOffRequest)
		request.Name = name
		fns[i] = func() (*onoffpb.OnOff, error) {
			return g.client.GetOnOff(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some hvacs failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeOnOff(slices.Values(allRes))
}

func (g *Group) UpdateOnOff(ctx context.Context, request *onoffpb.UpdateOnOffRequest) (*onoffpb.OnOff, error) {
	if g.readOnly {
		return nil, status.Errorf(codes.FailedPrecondition, "read-only")
	}
	fns := make([]func() (*onoffpb.OnOff, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*onoffpb.UpdateOnOffRequest)
		request.Name = name
		fns[i] = func() (*onoffpb.OnOff, error) {
			return g.client.UpdateOnOff(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some hvacs failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeOnOff(slices.Values(allRes))
}

func (g *Group) PullOnOff(request *onoffpb.PullOnOffRequest, server grpc.ServerStreamingServer[onoffpb.PullOnOffResponse]) error {
	if len(g.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no on off names")
	}

	type c struct {
		name string
		val  *onoffpb.OnOff
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range g.names {
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullOnOff(ctx, &onoffpb.PullOnOffRequest{Name: name})
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							err := chans.SendContext(ctx, changes, c{name: name, val: change.OnOff})
							if err != nil {
								return err
							}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetOnOff(ctx, &onoffpb.GetOnOffRequest{Name: name})
					if err != nil {
						return err
					}
					return chans.SendContext(ctx, changes, c{name: name, val: res})
				},
			), changes)
		})
	}

	group.Go(func() error {
		all := make(map[string]*onoffpb.OnOff, len(g.names))

		var last *onoffpb.OnOff
		eq := cmp.Equal()
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				all[change.name] = change.val

				values, err := mergeOnOff(maps.Values(all))
				if err != nil {
					return err
				}
				filter.Filter(values)
				if eq(last, values) {
					continue
				}
				last = values
				err = server.Send(&onoffpb.PullOnOffResponse{Changes: []*onoffpb.PullOnOffResponse_Change{
					{
						Name:       request.Name,
						ChangeTime: timestamppb.Now(),
						OnOff:      values,
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

func mergeOnOff(seq iter.Seq[*onoffpb.OnOff]) (*onoffpb.OnOff, error) {
	var state onoffpb.OnOff_State
	var got bool
	for v := range seq {
		got = true
		switch {
		case v.State == onoffpb.OnOff_STATE_UNSPECIFIED:
			// skip
		case state == onoffpb.OnOff_STATE_UNSPECIFIED:
			state = v.State
		case state != v.State:
			return nil, status.Errorf(codes.FailedPrecondition, "not all onoff devices have the same state")
		}
	}
	if !got {
		return nil, status.Errorf(codes.FailedPrecondition, "no onoff devices")
	}
	return &onoffpb.OnOff{
		State: state,
	}, nil
}
