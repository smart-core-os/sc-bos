package soundsensor

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/merge"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

type Group struct {
	soundsensorpb.UnimplementedSoundSensorApiServer
	client soundsensorpb.SoundSensorApiClient
	names  []string

	logger *zap.Logger
}

func (g *Group) GetSoundLevel(ctx context.Context, request *soundsensorpb.GetSoundLevelRequest) (*soundsensorpb.SoundLevel, error) {
	fns := make([]func() (*soundsensorpb.SoundLevel, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*soundsensorpb.GetSoundLevelRequest)
		request.Name = name
		fns[i] = func() (*soundsensorpb.SoundLevel, error) {
			return g.client.GetSoundLevel(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some sound sensors failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeSoundLevel(allRes)
}

func (g *Group) PullSoundLevel(request *soundsensorpb.PullSoundLevelRequest, server soundsensorpb.SoundSensorApi_PullSoundLevelServer) error {
	if len(g.names) == 0 {
		return status.Errorf(codes.FailedPrecondition, "zone has no sound sensor names")
	}

	type c struct {
		name string
		val  *soundsensorpb.SoundLevel
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	// get sound level from each of the named devices
	for _, name := range g.names {
		request := proto.Clone(request).(*soundsensorpb.PullSoundLevelRequest)
		request.Name = name
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullSoundLevel(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{name: request.Name, val: change.SoundLevel}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetSoundLevel(ctx, &soundsensorpb.GetSoundLevelRequest{Name: name, ReadMask: request.ReadMask})
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

	// merge all the changes into one soundLevel obj and send to server
	group.Go(func() error {
		// indexes reports which index in values each name has
		indexes := make(map[string]int, len(g.names))
		for i, name := range g.names {
			indexes[name] = i
		}
		values := make([]*soundsensorpb.SoundLevel, len(g.names))

		var last *soundsensorpb.SoundLevel
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change.val
				r, err := mergeSoundLevel(values)
				if err != nil {
					return err
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&soundsensorpb.PullSoundLevelResponse{Changes: []*soundsensorpb.PullSoundLevelResponse_Change{{
					Name:       request.Name,
					ChangeTime: timestamppb.Now(),
					SoundLevel: r,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func mergeSoundLevel(all []*soundsensorpb.SoundLevel) (*soundsensorpb.SoundLevel, error) {
	switch len(all) {
	case 0:
		return nil, status.Errorf(codes.FailedPrecondition, "zone has no sound sensor names")
	case 1:
		return all[0], nil
	default:
		out := &soundsensorpb.SoundLevel{}
		// SoundPressureLevel - use mean average
		if val, ok := merge.Mean(all, func(e *soundsensorpb.SoundLevel) (float32, bool) {
			if e == nil || e.SoundPressureLevel == nil {
				return 0, false
			}
			return *e.SoundPressureLevel, true
		}); ok {
			out.SoundPressureLevel = &val
		}
		return out, nil
	}
}

