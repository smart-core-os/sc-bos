package electric

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/merge"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

type Group struct {
	electricpb.UnimplementedElectricApiServer
	client electricpb.ElectricApiClient
	names  []string

	logger *zap.Logger
}

func (g *Group) GetDemand(ctx context.Context, request *electricpb.GetDemandRequest) (*electricpb.ElectricDemand, error) {
	fns := make([]func() (*electricpb.ElectricDemand, error), len(g.names))
	for i, name := range g.names {
		request := proto.Clone(request).(*electricpb.GetDemandRequest)
		request.Name = name
		fns[i] = func() (*electricpb.ElectricDemand, error) {
			return g.client.GetDemand(ctx, request)
		}
	}
	allRes, allErrs := run.Collect(ctx, run.DefaultConcurrency, fns...)

	err := multierr.Combine(allErrs...)
	if len(multierr.Errors(err)) == len(g.names) {
		return nil, err
	}

	if err != nil {
		if g.logger != nil {
			g.logger.Warn("some electrics failed to get", zap.Errors("errors", multierr.Errors(err)))
		}
	}
	return mergeDemand(allRes)
}

func (g *Group) PullDemand(request *electricpb.PullDemandRequest, server electricpb.ElectricApi_PullDemandServer) error {
	if len(g.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no electric names")
	}

	type c struct {
		name string
		val  *electricpb.ElectricDemand
	}
	changes := make(chan c)
	defer close(changes)

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range g.names {
		request := proto.Clone(request).(*electricpb.PullDemandRequest)
		request.Name = name
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- c) error {
					stream, err := g.client.PullDemand(ctx, request)
					if err != nil {
						return err
					}
					for {
						res, err := stream.Recv()
						if err != nil {
							return err
						}
						for _, change := range res.Changes {
							changes <- c{name: request.Name, val: change.Demand}
						}
					}
				},
				func(ctx context.Context, changes chan<- c) error {
					res, err := g.client.GetDemand(ctx, &electricpb.GetDemandRequest{Name: name, ReadMask: request.ReadMask})
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
		values := make([]*electricpb.ElectricDemand, len(g.names))

		var last *electricpb.ElectricDemand
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change.val
				r, err := mergeDemand(values)
				if err != nil {
					return err
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&electricpb.PullDemandResponse{Changes: []*electricpb.PullDemandResponse_Change{{
					Name:       request.Name,
					ChangeTime: timestamppb.Now(),
					Demand:     r,
				}}})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func mergeDemand(all []*electricpb.ElectricDemand) (*electricpb.ElectricDemand, error) {
	switch len(all) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no electric names")
	case 1:
		return all[0], nil
	default:
		out := &electricpb.ElectricDemand{}
		out.Current, _ = merge.Sum(all, func(e *electricpb.ElectricDemand) (float32, bool) {
			if e == nil {
				return 0, false
			}
			return e.Current, true
		})
		out.Rating, _ = merge.Sum(all, func(e *electricpb.ElectricDemand) (float32, bool) {
			if e == nil {
				return 0, false
			}
			return e.Rating, true
		})
		// Either all the voltages are the same or we can't set out.Voltage
		for _, e := range all {
			if e == nil || e.Voltage == nil {
				continue
			}
			if out.Voltage == nil {
				out.Voltage = e.Voltage
				continue
			}
			if *out.Voltage != *e.Voltage {
				// not all voltages are equal, so we can't set
				out.Voltage = nil
				break
			}
		}
		out.RealPower = merge.Ptr(merge.Sum(all, func(e *electricpb.ElectricDemand) (float32, bool) {
			if e == nil || e.RealPower == nil {
				return 0, false
			}
			return *e.RealPower, true
		}))
		out.ApparentPower = merge.Ptr(merge.Sum(all, func(e *electricpb.ElectricDemand) (float32, bool) {
			if e == nil || e.ApparentPower == nil {
				return 0, false
			}
			return *e.ApparentPower, true
		}))
		out.ReactivePower = merge.Ptr(merge.Sum(all, func(e *electricpb.ElectricDemand) (float32, bool) {
			if e == nil || e.ReactivePower == nil {
				return 0, false
			}
			return *e.ReactivePower, true
		}))
		return out, nil
	}
}
