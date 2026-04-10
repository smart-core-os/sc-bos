package occupancy

import (
	"context"
	// "log"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

type Group struct {
	occupancysensorpb.UnimplementedOccupancySensorApiServer
	client occupancysensorpb.OccupancySensorApiClient
	names  []string

	clients []occupancysensorpb.OccupancySensorApiClient // dedicated clients that don't use names for anything

	logger *zap.Logger
}

func (g *Group) GetOccupancy(ctx context.Context, request *occupancysensorpb.GetOccupancyRequest) (*occupancysensorpb.Occupancy, error) {
	fns := make([]func() (*occupancysensorpb.Occupancy, error), len(g.names)+len(g.clients))
	for i, name := range g.names {
		request := proto.Clone(request).(*occupancysensorpb.GetOccupancyRequest)
		request.Name = name
		fns[i] = run.TagError(name, func() (*occupancysensorpb.Occupancy, error) {
			return g.client.GetOccupancy(ctx, request)
		})
	}
	for i, client := range g.clients {
		fns[i+len(g.names)] = run.TagError("client", func() (*occupancysensorpb.Occupancy, error) {
			return client.GetOccupancy(ctx, request)
		})
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some occupancy sensors failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeOccupancy(allRes)
}

func (g *Group) PullOccupancy(request *occupancysensorpb.PullOccupancyRequest, server occupancysensorpb.OccupancySensorApi_PullOccupancyServer) error {
	if len(g.names) == 0 && len(g.clients) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no occupancy sensors")
	}

	// log.Printf("PullOccupancy(%v)", request.Name)
	// defer log.Printf("PullOccupancy(%v) done", request.Name)

	type c struct {
		index int
		val   *occupancysensorpb.Occupancy
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())

	// get occupancy from each of the named devices
	for i, name := range g.names {
		request := proto.Clone(request).(*occupancysensorpb.PullOccupancyRequest)
		request.Name = name
		index := i
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullOccupancy(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{index: index, val: change.Occupancy}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: name, ReadMask: request.ReadMask})
					if err != nil {
						return err
					}
					changes <- c{index: index, val: res}
					return nil
				}),
				changes,
			)
		})
	}

	// get occupancy from each of the dedicated clients
	for i, client := range g.clients {
		index := len(g.names) + i
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := client.PullOccupancy(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{index: index, val: change.Occupancy}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := client.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: request.Name, ReadMask: request.ReadMask})
					if err != nil {
						return err
					}
					changes <- c{index: index, val: res}
					return nil
				}),
				changes,
			)
		})
	}

	// merge all the changes into one occupancy and send to server
	group.Go(func() error {
		// indexes reports which index in values each name name has
		values := make([]*occupancysensorpb.Occupancy, len(g.names)+len(g.clients))

		var last *occupancysensorpb.Occupancy
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[change.index] = change.val
				r, err := mergeOccupancy(values)
				if err != nil {
					return err
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&occupancysensorpb.PullOccupancyResponse{Changes: []*occupancysensorpb.PullOccupancyResponse_Change{{
					Name:       request.Name,
					ChangeTime: timestamppb.Now(),
					Occupancy:  r,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func mergeOccupancy(all []*occupancysensorpb.Occupancy) (*occupancysensorpb.Occupancy, error) {
	switch len(all) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no occupancy sensor names")
	case 1:
		return all[0], nil
	default:
		out := &occupancysensorpb.Occupancy{}
		nilCount := 0
		occupiedCount := 0
		var earliestOccupiedTime, latestUnoccupiedTime time.Time
		for _, occupancy := range all {
			if occupancy == nil {
				nilCount++
				continue
			}

			out.PeopleCount += occupancy.PeopleCount

			switch occupancy.State {
			case occupancysensorpb.Occupancy_OCCUPIED:
				occupiedCount++

				// Recording the state change time takes our priority for occupied over unoccupied.
				// We do this by recording the earliest unoccupied time in out.StateChangeTime, and the earliest occupied time
				// in earliestOccupiedTime.
				// If after processing all the records we determine that we should be occupied then we swap out the state change time.
				if occupancy.StateChangeTime != nil {
					if earliestOccupiedTime.IsZero() || earliestOccupiedTime.After(occupancy.StateChangeTime.AsTime()) {
						earliestOccupiedTime = occupancy.StateChangeTime.AsTime()
					}
				}
			default:
				if occupancy.StateChangeTime != nil {
					if latestUnoccupiedTime.IsZero() || latestUnoccupiedTime.Before(occupancy.StateChangeTime.AsTime()) {
						latestUnoccupiedTime = occupancy.StateChangeTime.AsTime()
					}
				}
			}
		}
		if occupiedCount > 0 {
			out.State = occupancysensorpb.Occupancy_OCCUPIED
			if !earliestOccupiedTime.IsZero() {
				out.StateChangeTime = timestamppb.New(earliestOccupiedTime)
			}
		} else {
			if len(all) > nilCount {
				out.State = occupancysensorpb.Occupancy_UNOCCUPIED
				out.Confidence = float64(nilCount) / float64(len(all))
				if !latestUnoccupiedTime.IsZero() {
					out.StateChangeTime = timestamppb.New(latestUnoccupiedTime)
				}
			}
		}
		return out, nil
	}
}
