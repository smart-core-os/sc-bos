package enterleave

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/enterleave/config"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
	"github.com/smart-core-os/sc-golang/pkg/masks"
)

type Group struct {
	traits.UnimplementedEnterLeaveSensorApiServer

	enterLeaveClients []traits.EnterLeaveSensorApiClient

	logger *zap.Logger
}

type feature struct {
	*service.Service[config.Root]
	announcer *node.ReplaceAnnouncer
	devices   *zone.Devices
	clients   node.ClientConner
	logger    *zap.Logger
}

var Feature = zone.FactoryFunc(func(services zone.Services) service.Lifecycle {
	services.Logger = services.Logger.Named("enterleave")
	f := &feature{
		announcer: node.NewReplaceAnnouncer(services.Node),
		devices:   services.Devices,
		clients:   services.Node,
		logger:    services.Logger,
	}
	f.Service = service.New(service.MonoApply(f.applyConfig))
	return f
})

func (g *Group) PullEnterLeaveEvents(request *traits.PullEnterLeaveEventsRequest, server traits.EnterLeaveSensorApi_PullEnterLeaveEventsServer) error {
	if len(g.enterLeaveClients) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no enterleave sensors")
	}

	type c struct {
		index int
		val   *traits.EnterLeaveEvent
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())

	// get enterleave from each of the dedicated clients
	for i, client := range g.enterLeaveClients {
		client := client
		index := i
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := client.PullEnterLeaveEvents(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{index: index, val: change.EnterLeaveEvent}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := client.GetEnterLeaveEvent(ctx, &traits.GetEnterLeaveEventRequest{Name: request.Name, ReadMask: request.ReadMask})
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

	// merge all the changes into one EnterLeaveEvent and send to server
	group.Go(func() error {
		// indexes reports which index in values each name name has
		values := make([]*traits.EnterLeaveEvent, len(g.enterLeaveClients))

		var last *traits.EnterLeaveEvent
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[change.index] = change.val
				r, err := mergeEnterLeave(values)
				if err != nil {
					return err
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&traits.PullEnterLeaveEventsResponse{Changes: []*traits.PullEnterLeaveEventsResponse_Change{{
					Name:            request.Name,
					ChangeTime:      timestamppb.Now(),
					EnterLeaveEvent: r,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func (g *Group) GetEnterLeaveEvent(ctx context.Context, request *traits.GetEnterLeaveEventRequest) (*traits.EnterLeaveEvent, error) {
	fns := make([]func() (*traits.EnterLeaveEvent, error), len(g.enterLeaveClients))

	for i, client := range g.enterLeaveClients {
		fns[i] = func() (*traits.EnterLeaveEvent, error) {
			return client.GetEnterLeaveEvent(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.enterLeaveClients) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some enterleave sensors failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeEnterLeave(allRes)
}

func mergeEnterLeave(all []*traits.EnterLeaveEvent) (*traits.EnterLeaveEvent, error) {

	if len(all) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "zone has no enterleave sensor names")
	} else if len(all) == 1 {
		return all[0], nil
	}

	enterTotal := int32(0)
	leaveTotal := int32(0)

	for _, e := range all {

		if e == nil {
			continue
		}

		if e.EnterTotal != nil {
			enterTotal += *e.EnterTotal
		}

		if e.LeaveTotal != nil {
			leaveTotal += *e.LeaveTotal
		}
	}

	return &traits.EnterLeaveEvent{
		EnterTotal: &enterTotal,
		LeaveTotal: &leaveTotal,
	}, nil
}

func (g *Group) ResetEnterLeaveTotals(ctx context.Context, request *traits.ResetEnterLeaveTotalsRequest) (*traits.ResetEnterLeaveTotalsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
